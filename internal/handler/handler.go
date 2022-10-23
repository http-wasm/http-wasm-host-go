// Package internalhandler is not named handler as doing so interferes with
// godoc links for the api handler package.
package internalhandler

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/internal"
)

// Middleware implements the http-wasm handler ABI.
// It is scoped to a single guest binary.
type Middleware interface {
	// Handle handles a request by calling handler.FuncHandle on the guest.
	Handle(ctx context.Context) error

	// Features are the features enabled while initializing the guest. This
	// value won't change per-request.
	Features() handler.Features

	api.Closer
}

var _ Middleware = (*middleware)(nil)

type middleware struct {
	host                    handler.Host
	runtime                 wazero.Runtime
	hostModule, guestModule wazero.CompiledModule
	newNamespace            httpwasm.NewNamespace
	moduleConfig            wazero.ModuleConfig
	guestConfig             []byte
	logger                  api.Logger
	pool                    sync.Pool
	features                handler.Features
}

func (m *middleware) Features() handler.Features {
	return m.features
}

func NewMiddleware(ctx context.Context, guest []byte, host handler.Host, options ...httpwasm.Option) (Middleware, error) {
	o := &internal.WazeroOptions{
		NewRuntime:   internal.DefaultRuntime,
		NewNamespace: internal.DefaultNamespace,
		ModuleConfig: wazero.NewModuleConfig(),
		Logger:       api.NoopLogger{},
	}
	for _, option := range options {
		option(o)
	}

	wr, err := o.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating middleware: %w", err)
	}

	m := &middleware{
		host:         host,
		runtime:      wr,
		newNamespace: o.NewNamespace,
		moduleConfig: o.ModuleConfig,
		guestConfig:  o.GuestConfig,
		logger:       o.Logger,
	}

	if m.hostModule, err = m.compileHost(ctx); err != nil {
		_ = m.Close(ctx)
		return nil, err
	}

	if m.guestModule, err = m.compileGuest(ctx, guest); err != nil {
		_ = m.Close(ctx)
		return nil, err
	}

	if g, err := m.newGuest(ctx); err != nil {
		_ = m.Close(ctx)
		return nil, err
	} else {
		m.pool.Put(g)
	}

	return m, nil
}

func (m *middleware) compileGuest(ctx context.Context, wasm []byte) (wazero.CompiledModule, error) {
	if guest, err := m.runtime.CompileModule(ctx, wasm); err != nil {
		return nil, fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if handle, ok := guest.ExportedFunctions()[handler.FuncHandle]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export func[%s]", handler.FuncHandle)
	} else if len(handle.ParamTypes()) != 0 || len(handle.ResultTypes()) != 0 {
		return nil, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be nullary", handler.FuncHandle)
	} else if _, ok = guest.ExportedMemories()[api.Memory]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export memory[%s]", api.Memory)
	} else {
		return guest, nil
	}
}

// Handle implements Middleware.Handle
func (m *middleware) Handle(ctx context.Context) error {
	poolG := m.pool.Get()
	if poolG == nil {
		g, err := m.newGuest(ctx)
		if err != nil {
			return err
		}
		poolG = g
	}
	g := poolG.(*guest)
	defer m.pool.Put(g)
	s := &requestState{features: m.features}
	defer s.Close()
	ctx = context.WithValue(ctx, requestStateKey{}, s)
	return g.handle(ctx)
}

// next calls the same function as documented on handler.Host.
func (m *middleware) next(ctx context.Context) {
	s := requestStateFromContext(ctx)
	if s.calledNext {
		panic("already called next")
	}
	s.calledNext = true
	_ = s.closeRequest()
	m.host.Next(ctx)
}

// Close implements api.Closer
func (m *middleware) Close(ctx context.Context) error {
	// We don't have to close any guests as the middleware will close it.
	return m.runtime.Close(ctx)
}

type guest struct {
	ns         wazero.Namespace
	guest      wazeroapi.Module
	handleFunc wazeroapi.Function
}

func (m *middleware) newGuest(ctx context.Context) (*guest, error) {
	ns, err := m.newNamespace(ctx, m.runtime)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating namespace: %w", err)
	}

	// Note: host modules don't use configuration
	_, err = ns.InstantiateModule(ctx, m.hostModule, wazero.NewModuleConfig())
	if err != nil {
		_ = ns.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating host: %w", err)
	}

	g, err := ns.InstantiateModule(ctx, m.guestModule, m.moduleConfig)
	if err != nil {
		_ = ns.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &guest{ns: ns, guest: g, handleFunc: g.ExportedFunction(handler.FuncHandle)}, nil
}

// handle calls the WebAssembly guest function handler.FuncHandle.
func (g *guest) handle(ctx context.Context) (err error) {
	_, err = g.handleFunc.Call(ctx)
	return
}

// enableFeatures implements the WebAssembly host function handler.FuncEnableFeatures.
func (m *middleware) enableFeatures(ctx context.Context, features handler.Features) (enabled handler.Features) {
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.features = m.host.EnableFeatures(ctx, s.features.WithEnabled(features))
		enabled = s.features
	} else {
		m.features = m.host.EnableFeatures(ctx, m.features.WithEnabled(features))
		enabled = m.features
	}
	return enabled
}

// getConfig implements the WebAssembly host function handler.FuncGetConfig.
func (m *middleware) getConfig(ctx context.Context, mod wazeroapi.Module,
	buf uint32, bufLimit handler.BufLimit) (len uint32) {
	return writeIfUnderLimit(ctx, mod, buf, bufLimit, m.guestConfig)
}

// log implements the WebAssembly host function handler.FuncLogEnabled.
func (m *middleware) logEnabled(level api.LogLevel) uint32 {
	if m.logger.IsEnabled(level) {
		return 1
	}
	return 0
}

// log implements the WebAssembly host function handler.FuncLog.
func (m *middleware) log(ctx context.Context, mod wazeroapi.Module,
	level api.LogLevel, message, messageLen uint32) {
	if !m.logger.IsEnabled(level) {
		return
	}
	var msg string
	if messageLen > 0 {
		msg = mustReadString(ctx, mod.Memory(), "message", message, messageLen)
	}
	m.logger.Log(ctx, level, msg)
}

// getMethod implements the WebAssembly host function handler.FuncGetMethod.
func (m *middleware) getMethod(ctx context.Context, mod wazeroapi.Module,
	buf uint32, bufLimit handler.BufLimit) (len uint32) {
	method := m.host.GetMethod(ctx)
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, method)
}

// getHeader implements the WebAssembly host function
// handler.FuncSetMethod.
func (m *middleware) setMethod(ctx context.Context, mod wazeroapi.Module,
	method, methodLen uint32) {
	_ = mustBeforeNext(ctx, "set", "method")

	var p string
	if methodLen == 0 {
		panic("HTTP method cannot be empty")
	}
	p = mustReadString(ctx, mod.Memory(), "method", method, methodLen)
	m.host.SetMethod(ctx, p)
}

// getURI implements the WebAssembly host function handler.FuncGetURI.
func (m *middleware) getURI(ctx context.Context, mod wazeroapi.Module,
	buf uint32, bufLimit handler.BufLimit) (len uint32) {
	uri := m.host.GetURI(ctx)
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, uri)
}

// getHeader implements the WebAssembly host function
// handler.FuncSetURI.
func (m *middleware) setURI(ctx context.Context, mod wazeroapi.Module,
	uri, uriLen uint32) {
	var p string
	if uriLen > 0 { // overwrite with empty is supported
		p = mustReadString(ctx, mod.Memory(), "uri", uri, uriLen)
	}
	m.host.SetURI(ctx, p)
}

// getProtocolVersion implements the WebAssembly host function
// handler.FuncGetProtocolVersion.
func (m *middleware) getProtocolVersion(ctx context.Context, mod wazeroapi.Module,
	buf uint32, bufLimit handler.BufLimit) uint32 {
	protocolVersion := m.host.GetProtocolVersion(ctx)
	if len(protocolVersion) == 0 {
		panic("HTTP protocol version cannot be empty")
	}
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, protocolVersion)
}

// getHeaderNames implements the WebAssembly host function
// handler.FuncGetHeaderNames.
func (m *middleware) getHeaderNames(ctx context.Context, mod wazeroapi.Module,
	kind handler.HeaderKind, buf uint32, bufLimit handler.BufLimit) (countLen handler.CountLen) {
	var names []string
	switch kind {
	case handler.HeaderKindRequest:
		names = m.host.GetRequestHeaderNames(ctx)
	case handler.HeaderKindRequestTrailers:
		names = m.host.GetRequestTrailerNames(ctx)
	case handler.HeaderKindResponse:
		names = m.host.GetResponseHeaderNames(ctx)
	case handler.HeaderKindResponseTrailers:
		names = m.host.GetResponseTrailerNames(ctx)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
	return writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, names)
}

// getHeaderValues implements the WebAssembly host function
// handler.FuncGetHeaderValues.
func (m *middleware) getHeaderValues(ctx context.Context, mod wazeroapi.Module,
	kind handler.HeaderKind, name, nameLen, buf uint32, bufLimit handler.BufLimit) (countLen handler.CountLen) {
	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)

	var values []string
	switch kind {
	case handler.HeaderKindRequest:
		values = m.host.GetRequestHeaderValues(ctx, n)
	case handler.HeaderKindRequestTrailers:
		values = m.host.GetRequestTrailerValues(ctx, n)
	case handler.HeaderKindResponse:
		values = m.host.GetResponseHeaderValues(ctx, n)
	case handler.HeaderKindResponseTrailers:
		values = m.host.GetResponseTrailerValues(ctx, n)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
	return writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, values)
}

// setHeaderValue implements the WebAssembly host function
// handler.FuncSetHeaderValue.
func (m *middleware) setHeaderValue(ctx context.Context, mod wazeroapi.Module,
	kind handler.HeaderKind, name, nameLen, value, valueLen uint32) {
	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "set", kind)
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v := mustReadString(ctx, mod.Memory(), "value", value, valueLen)

	switch kind {
	case handler.HeaderKindRequest:
		m.host.SetRequestHeaderValue(ctx, n, v)
	case handler.HeaderKindRequestTrailers:
		m.host.SetRequestTrailerValue(ctx, n, v)
	case handler.HeaderKindResponse:
		m.host.SetResponseHeaderValue(ctx, n, v)
	case handler.HeaderKindResponseTrailers:
		m.host.SetResponseTrailerValue(ctx, n, v)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// addHeaderValue implements the WebAssembly host function
// handler.FuncAddHeader.
func (m *middleware) addHeaderValue(ctx context.Context, mod wazeroapi.Module,
	kind handler.HeaderKind, name, nameLen, value, valueLen uint32) {
	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "add", kind)
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v := mustReadString(ctx, mod.Memory(), "value", value, valueLen)

	switch kind {
	case handler.HeaderKindRequest:
		m.host.AddRequestHeaderValue(ctx, n, v)
	case handler.HeaderKindRequestTrailers:
		m.host.AddRequestTrailerValue(ctx, n, v)
	case handler.HeaderKindResponse:
		m.host.AddResponseHeaderValue(ctx, n, v)
	case handler.HeaderKindResponseTrailers:
		m.host.AddResponseTrailerValue(ctx, n, v)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// removeHeader implements the WebAssembly host function
// handler.FuncRemoveHeader.
func (m *middleware) removeHeader(ctx context.Context, mod wazeroapi.Module,
	kind handler.HeaderKind, name, nameLen uint32) {
	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "remove", kind)
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)

	switch kind {
	case handler.HeaderKindRequest:
		m.host.RemoveRequestHeader(ctx, n)
	case handler.HeaderKindRequestTrailers:
		m.host.RemoveRequestTrailer(ctx, n)
	case handler.HeaderKindResponse:
		m.host.RemoveResponseHeader(ctx, n)
	case handler.HeaderKindResponseTrailers:
		m.host.RemoveResponseTrailer(ctx, n)
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// readBody implements the WebAssembly host function handler.FuncReadBody.
func (m *middleware) readBody(ctx context.Context, mod wazeroapi.Module,
	kind handler.BodyKind, buf uint32, bufLimit handler.BufLimit) (eofLen handler.EOFLen) {

	var r io.ReadCloser
	switch kind {
	case handler.BodyKindRequest:
		s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferRequest, "read", "request body")
		// Lazy create the reader.
		r = s.requestBodyReader
		if r == nil {
			r = m.host.RequestBodyReader(ctx)
			s.requestBodyReader = r
		}
	case handler.BodyKindResponse:
		s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "read", "response body")
		// Lazy create the reader.
		r = s.responseBodyReader
		if r == nil {
			r = m.host.ResponseBodyReader(ctx)
			s.responseBodyReader = r
		}
	default:
		panic("unsupported body kind: " + strconv.Itoa(int(kind)))
	}

	return readBody(ctx, mod, buf, bufLimit, r)
}

// writeBody implements the WebAssembly host function handler.FuncWriteBody.
func (m *middleware) writeBody(ctx context.Context, mod wazeroapi.Module,
	kind handler.BodyKind, buf, bufLen uint32) {

	var w io.Writer
	switch kind {
	case handler.BodyKindRequest:
		s := mustBeforeNext(ctx, "write", "request body")
		// Lazy create the writer.
		w = s.requestBodyWriter
		if w == nil {
			w = m.host.RequestBodyWriter(ctx)
			s.requestBodyWriter = w
		}
	case handler.BodyKindResponse:
		s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "write", "response body")
		// Lazy create the writer.
		w = s.responseBodyWriter
		if w == nil {
			w = m.host.ResponseBodyWriter(ctx)
			s.responseBodyWriter = w
		}
	default:
		panic("unsupported body kind: " + strconv.Itoa(int(kind)))
	}

	writeBody(ctx, mod, buf, bufLen, w)
}

func writeBody(ctx context.Context, mod wazeroapi.Module, buf, bufLen uint32, w io.Writer) {
	// buf_len 0 means to overwrite with nothing
	var b []byte
	if bufLen > 0 {
		b = mustRead(ctx, mod.Memory(), "body", buf, bufLen)
	}
	if _, err := w.Write(b); err != nil { // Write errs if it can't write n bytes
		panic(fmt.Errorf("error writing body: %w", err))
	}
}

// getStatusCode implements the WebAssembly host function
// handler.FuncGetStatusCode.
func (m *middleware) getStatusCode(ctx context.Context) uint32 {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "get", "status code")

	return m.host.GetStatusCode(ctx)
}

// setStatusCode implements the WebAssembly host function
// handler.FuncSetStatusCode.
func (m *middleware) setStatusCode(ctx context.Context, statusCode uint32) {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "set", "status code")

	m.host.SetStatusCode(ctx, statusCode)
}

func readBody(ctx context.Context, mod wazeroapi.Module, buf uint32, bufLimit handler.BufLimit, r io.Reader) (eofLen uint64) {
	// buf_limit 0 serves no purpose as implementations won't return EOF on it.
	if bufLimit == 0 {
		panic(fmt.Errorf("buf_limit==0 reading body"))
	}

	// Allocate a buf to write into directly
	b := mustRead(ctx, mod.Memory(), "body", buf, bufLimit)

	// Attempt to fill the buffer until an error occurs. Notably, this works
	// around a full read not returning EOF until the
	var err error
	n := uint32(0)
	for n < bufLimit && err == nil {
		var nn int
		nn, err = r.Read(b[n:])
		n += uint32(nn)
	}

	if err == nil {
		return uint64(n) // Not EOF
	} else if err == io.EOF { // EOF is by contract, so can't be wrapped
		return uint64(1<<32) | uint64(n)
	} else {
		panic(fmt.Errorf("error reading body: %w", err))
	}
}

func mustBeforeNext(ctx context.Context, op, kind string) (s *requestState) {
	if s = requestStateFromContext(ctx); s.calledNext {
		panic(fmt.Errorf("can't %s %s after next handler", op, kind))
	}
	return
}

func mustBeforeNextOrFeature(ctx context.Context, feature handler.Features, op, kind string) (s *requestState) {
	if s = requestStateFromContext(ctx); !s.calledNext {
		// Assume this is serving a response from the guest.
	} else if s.features.IsEnabled(feature) {
		// Assume the guest is overwriting the response from next.
	} else {
		panic(fmt.Errorf("can't %s %s after next handler unless %s is enabled",
			op, kind, feature))
	}
	return
}

func (m *middleware) compileHost(ctx context.Context) (wazero.CompiledModule, error) {
	if compiled, err := m.runtime.NewHostModuleBuilder(handler.HostModule).
		ExportFunction(handler.FuncEnableFeatures, m.enableFeatures,
			handler.FuncEnableFeatures, "features").
		ExportFunction(handler.FuncGetConfig, m.getConfig,
			handler.FuncGetConfig, "buf", "buf_limit").
		ExportFunction(handler.FuncLogEnabled, m.logEnabled,
			handler.FuncLogEnabled, "level").
		ExportFunction(handler.FuncLog, m.log,
			handler.FuncLog, "level", "message", "message_len").
		ExportFunction(handler.FuncGetMethod, m.getMethod,
			handler.FuncGetMethod, "buf", "buf_limit").
		ExportFunction(handler.FuncSetMethod, m.setMethod,
			handler.FuncSetMethod, "method", "method_len").
		ExportFunction(handler.FuncGetURI, m.getURI,
			handler.FuncGetURI, "buf", "buf_limit").
		ExportFunction(handler.FuncSetURI, m.setURI,
			handler.FuncSetURI, "uri", "uri_len").
		ExportFunction(handler.FuncGetProtocolVersion, m.getProtocolVersion,
			handler.FuncGetProtocolVersion, "buf", "buf_limit").
		ExportFunction(handler.FuncGetHeaderNames, m.getHeaderNames,
			handler.FuncGetHeaderNames, "kind", "buf", "buf_limit").
		ExportFunction(handler.FuncGetHeaderValues, m.getHeaderValues,
			handler.FuncGetHeaderValues, "kind", "name", "name_len", "buf", "buf_limit").
		ExportFunction(handler.FuncSetHeaderValue, m.setHeaderValue,
			handler.FuncSetHeaderValue, "kind", "name", "name_len", "value", "value_len").
		ExportFunction(handler.FuncAddHeaderValue, m.addHeaderValue,
			handler.FuncAddHeaderValue, "kind", "name", "name_len", "value", "value_len").
		ExportFunction(handler.FuncRemoveHeader, m.removeHeader,
			handler.FuncRemoveHeader, "kind", "name", "name_len").
		ExportFunction(handler.FuncReadBody, m.readBody,
			handler.FuncReadBody, "kind", "buf", "buf_limit").
		ExportFunction(handler.FuncWriteBody, m.writeBody,
			handler.FuncWriteBody, "kind", "body", "body_len").
		ExportFunction(handler.FuncNext, m.next,
			handler.FuncNext).
		ExportFunction(handler.FuncGetStatusCode, m.getStatusCode,
			handler.FuncGetStatusCode).
		ExportFunction(handler.FuncSetStatusCode, m.setStatusCode,
			handler.FuncSetStatusCode, "status_code").
		Compile(ctx); err != nil {
		return nil, fmt.Errorf("wasm: error compiling host: %w", err)
	} else {
		return compiled, nil
	}
}

func mustHeaderMutable(ctx context.Context, op string, kind handler.HeaderKind) {
	switch kind {
	case handler.HeaderKindRequest:
		_ = mustBeforeNext(ctx, op, "request header")
	case handler.HeaderKindRequestTrailers:
		_ = mustBeforeNext(ctx, op, "request trailer")
	case handler.HeaderKindResponse:
		_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, op, "response header")
	case handler.HeaderKindResponseTrailers:
		_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, op, "response trailer")
	default:
		panic("unsupported header kind: " + strconv.Itoa(int(kind)))
	}
}

// mustReadString is a convenience function that casts mustRead
func mustReadString(ctx context.Context, mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) string {
	if byteCount == 0 {
		return ""
	}
	return string(mustRead(ctx, mem, fieldName, offset, byteCount))
}

var emptyBody = make([]byte, 0)

// mustRead is like api.Memory except that it panics if the offset and byteCount are out of range.
func mustRead(ctx context.Context, mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) []byte {
	if byteCount == 0 {
		return emptyBody
	}
	buf, ok := mem.Read(ctx, offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}
	return buf
}

func writeIfUnderLimit(ctx context.Context, mod wazeroapi.Module, offset, limit handler.BufLimit, v []byte) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mod.Memory().Write(ctx, offset, v)
	return
}

func writeStringIfUnderLimit(ctx context.Context, mod wazeroapi.Module, offset, limit handler.BufLimit, v string) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mod.Memory().WriteString(ctx, offset, v)
	return
}
