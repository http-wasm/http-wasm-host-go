package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

// Middleware implements the http-wasm handler ABI.
// It is scoped to a single guest binary.
type Middleware interface {
	// HandleRequest handles a request by calling handler.FuncHandleRequest on
	// the guest.
	//
	// Note: If the handler.CtxNext is returned with `next=1`, you must call
	// HandleResponse.
	HandleRequest(ctx context.Context) (outCtx context.Context, ctxNext handler.CtxNext, err error)

	// HandleResponse handles a response by calling handler.FuncHandleResponse
	// on the guest. This is only called when HandleRequest returns
	// handler.CtxNext with `next=1`.
	//
	// The ctx and ctxNext parameters are those returned from HandleRequest.
	// Specifically, the handler.CtxNext "ctx" field is passed as `reqCtx`.
	// The err parameter is nil unless the host erred processing the next
	// handler.
	HandleResponse(ctx context.Context, reqCtx uint32, err error) error

	// Features are the features enabled while initializing the guest. This
	// value won't change per-request.
	Features() handler.Features

	api.Closer
}

var _ Middleware = (*middleware)(nil)

type middleware struct {
	host            handler.Host
	runtime         wazero.Runtime
	guestModule     wazero.CompiledModule
	moduleConfig    wazero.ModuleConfig
	guestConfig     []byte
	logger          api.Logger
	pool            sync.Pool
	features        handler.Features
	instanceCounter uint64
}

func (m *middleware) Features() handler.Features {
	return m.features
}

func NewMiddleware(ctx context.Context, guest []byte, host handler.Host, opts ...Option) (Middleware, error) {
	o := &options{
		newRuntime:   DefaultRuntime,
		moduleConfig: wazero.NewModuleConfig(),
		logger:       api.NoopLogger{},
	}
	for _, opt := range opts {
		opt(o)
	}

	wr, err := o.newRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating middleware: %w", err)
	}

	m := &middleware{
		host:         host,
		runtime:      wr,
		moduleConfig: o.moduleConfig,
		guestConfig:  o.guestConfig,
		logger:       o.logger,
	}

	if m.guestModule, err = m.compileGuest(ctx, guest); err != nil {
		_ = wr.Close(ctx)
		return nil, err
	}

	// Detect and handle any host imports or lack thereof.
	imports := detectImports(m.guestModule.ImportedFunctions())
	switch {
	case imports&importWasiP1 != 0:
		if _, err = wasi_snapshot_preview1.Instantiate(ctx, m.runtime); err != nil {
			_ = wr.Close(ctx)
			return nil, fmt.Errorf("wasm: error instantiating wasi: %w", err)
		}
		fallthrough // proceed to configure any http_handler imports
	case imports&importHttpHandler != 0:
		if _, err = m.instantiateHost(ctx); err != nil {
			_ = wr.Close(ctx)
			return nil, fmt.Errorf("wasm: error instantiating host: %w", err)
		}
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	if g, err := m.newGuest(ctx); err != nil {
		_ = wr.Close(ctx)
		return nil, err
	} else {
		m.pool.Put(g)
	}

	return m, nil
}

func (m *middleware) compileGuest(ctx context.Context, wasm []byte) (wazero.CompiledModule, error) {
	if guest, err := m.runtime.CompileModule(ctx, wasm); err != nil {
		return nil, fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if handleRequest, ok := guest.ExportedFunctions()[handler.FuncHandleRequest]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export func[%s]", handler.FuncHandleRequest)
	} else if len(handleRequest.ParamTypes()) != 0 || !bytes.Equal(handleRequest.ResultTypes(), []wazeroapi.ValueType{wazeroapi.ValueTypeI64}) {
		return nil, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be () -> (i64)", handler.FuncHandleRequest)
	} else if handleResponse, ok := guest.ExportedFunctions()[handler.FuncHandleResponse]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export func[%s]", handler.FuncHandleResponse)
	} else if !bytes.Equal(handleResponse.ParamTypes(), []wazeroapi.ValueType{wazeroapi.ValueTypeI32, wazeroapi.ValueTypeI32}) || len(handleResponse.ResultTypes()) != 0 {
		return nil, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be (i32, 32) -> ()", handler.FuncHandleResponse)
	} else if _, ok = guest.ExportedMemories()[api.Memory]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export memory[%s]", api.Memory)
	} else {
		return guest, nil
	}
}

// HandleRequest implements Middleware.HandleRequest
func (m *middleware) HandleRequest(ctx context.Context) (outCtx context.Context, ctxNext handler.CtxNext, err error) {
	g, guestErr := m.getOrCreateGuest(ctx)
	if guestErr != nil {
		err = guestErr
		return
	}

	s := &requestState{features: m.features, putPool: m.pool.Put, g: g}
	defer func() {
		if ctxNext != 0 { // will call the next handler
			if closeErr := s.closeRequest(); err == nil {
				err = closeErr
			}
		} else { // guest errored or returned the response
			if closeErr := s.Close(); err == nil {
				err = closeErr
			}
		}
	}()

	outCtx = context.WithValue(ctx, requestStateKey{}, s)
	ctxNext, err = g.handleRequest(outCtx)
	return
}

func (m *middleware) getOrCreateGuest(ctx context.Context) (*guest, error) {
	poolG := m.pool.Get()
	if poolG == nil {
		if g, createErr := m.newGuest(ctx); createErr != nil {
			return nil, createErr
		} else {
			poolG = g
		}
	}
	return poolG.(*guest), nil
}

// HandleResponse implements Middleware.HandleResponse
func (m *middleware) HandleResponse(ctx context.Context, reqCtx uint32, hostErr error) error {
	s := requestStateFromContext(ctx)
	defer s.Close()
	s.afterNext = true

	return s.g.handleResponse(ctx, reqCtx, hostErr)
}

// Close implements api.Closer
func (m *middleware) Close(ctx context.Context) error {
	// We don't have to close any guests as the middleware will close it.
	return m.runtime.Close(ctx)
}

type guest struct {
	guest            wazeroapi.Module
	handleRequestFn  wazeroapi.Function
	handleResponseFn wazeroapi.Function
}

func (m *middleware) newGuest(ctx context.Context) (*guest, error) {
	moduleName := fmt.Sprintf("%d", atomic.AddUint64(&m.instanceCounter, 1))

	g, err := m.runtime.InstantiateModule(ctx, m.guestModule, m.moduleConfig.WithName(moduleName))
	if err != nil {
		_ = m.runtime.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &guest{
		guest:            g,
		handleRequestFn:  g.ExportedFunction(handler.FuncHandleRequest),
		handleResponseFn: g.ExportedFunction(handler.FuncHandleResponse),
	}, nil
}

// handleRequest calls the WebAssembly guest function handler.FuncHandleRequest.
func (g *guest) handleRequest(ctx context.Context) (ctxNext handler.CtxNext, err error) {
	if results, guestErr := g.handleRequestFn.Call(ctx); guestErr != nil {
		err = guestErr
	} else {
		ctxNext = handler.CtxNext(results[0])
	}
	return
}

// handleResponse calls the WebAssembly guest function handler.FuncHandleResponse.
func (g *guest) handleResponse(ctx context.Context, reqCtx uint32, err error) error {
	wasError := uint64(0)
	if err != nil {
		wasError = 1
	}
	_, err = g.handleResponseFn.Call(ctx, uint64(reqCtx), wasError)
	return err
}

// enableFeatures implements the WebAssembly host function handler.FuncEnableFeatures.
func (m *middleware) enableFeatures(ctx context.Context, stack []uint64) {
	features := handler.Features(stack[0])

	var enabled handler.Features
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.features = m.host.EnableFeatures(ctx, s.features.WithEnabled(features))
		enabled = s.features
	} else {
		m.features = m.host.EnableFeatures(ctx, m.features.WithEnabled(features))
		enabled = m.features
	}

	stack[0] = uint64(enabled)
}

// getConfig implements the WebAssembly host function handler.FuncGetConfig.
func (m *middleware) getConfig(_ context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	configLen := writeIfUnderLimit(mod.Memory(), buf, bufLimit, m.guestConfig)

	stack[0] = uint64(configLen)
}

// log implements the WebAssembly host function handler.FuncLogEnabled.
func (m *middleware) logEnabled(_ context.Context, stack []uint64) {
	level := api.LogLevel(stack[0])
	if m.logger.IsEnabled(level) {
		stack[0] = 1 // true
	} else {
		stack[0] = 0 // false
	}
}

// log implements the WebAssembly host function handler.FuncLog.
func (m *middleware) log(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	level := api.LogLevel(params[0])
	message := uint32(params[1])
	messageLen := uint32(params[2])

	if !m.logger.IsEnabled(level) {
		return
	}
	var msg string
	if messageLen > 0 {
		msg = mustReadString(mod.Memory(), "message", message, messageLen)
	}
	m.logger.Log(ctx, level, msg)
}

// getMethod implements the WebAssembly host function handler.FuncGetMethod.
func (m *middleware) getMethod(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	method := m.host.GetMethod(ctx)
	methodLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, method)

	stack[0] = uint64(methodLen)
}

// getHeader implements the WebAssembly host function handler.FuncSetMethod.
func (m *middleware) setMethod(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	method := uint32(params[0])
	methodLen := uint32(params[1])

	_ = mustBeforeNext(ctx, "set", "method")

	var p string
	if methodLen == 0 {
		panic("HTTP method cannot be empty")
	}
	p = mustReadString(mod.Memory(), "method", method, methodLen)
	m.host.SetMethod(ctx, p)
}

// getURI implements the WebAssembly host function handler.FuncGetURI.
func (m *middleware) getURI(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	uri := m.host.GetURI(ctx)
	uriLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, uri)

	stack[0] = uint64(uriLen)
}

// setURI implements the WebAssembly host function handler.FuncSetURI.
func (m *middleware) setURI(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	uri := uint32(params[0])
	uriLen := uint32(params[1])

	_ = mustBeforeNext(ctx, "set", "uri")

	var p string
	if uriLen > 0 { // overwrite with empty is supported
		p = mustReadString(mod.Memory(), "uri", uri, uriLen)
	}
	m.host.SetURI(ctx, p)
}

// getProtocolVersion implements the WebAssembly host function
// handler.FuncGetProtocolVersion.
func (m *middleware) getProtocolVersion(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	protocolVersion := m.host.GetProtocolVersion(ctx)
	if len(protocolVersion) == 0 {
		panic("HTTP protocol version cannot be empty")
	}
	protocolVersionLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, protocolVersion)

	stack[0] = uint64(protocolVersionLen)
}

// getHeaderNames implements the WebAssembly host function
// handler.FuncGetHeaderNames.
func (m *middleware) getHeaderNames(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handler.HeaderKind(stack[0])
	buf := uint32(stack[1])
	bufLimit := handler.BufLimit(stack[2])

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

	// TODO: This will allocate new strings all the time. It could be optimized
	// by having writeNULTerminated directly lowercase while writing instead for
	// this code path.
	for i := range names {
		names[i] = strings.ToLower(names[i])
	}

	countLen := writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, names)

	stack[0] = countLen
}

// getHeaderValues implements the WebAssembly host function
// handler.FuncGetHeaderValues.
func (m *middleware) getHeaderValues(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handler.HeaderKind(stack[0])
	name := uint32(stack[1])
	nameLen := uint32(stack[2])
	buf := uint32(stack[3])
	bufLimit := handler.BufLimit(stack[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(mod.Memory(), "name", name, nameLen)

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
	countLen := writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, values)

	stack[0] = countLen
}

// setHeaderValue implements the WebAssembly host function
// handler.FuncSetHeaderValue.
func (m *middleware) setHeaderValue(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handler.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])
	value := uint32(params[3])
	valueLen := uint32(params[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "set", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)
	v := mustReadString(mod.Memory(), "value", value, valueLen)

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
// handler.FuncAddHeaderValue.
func (m *middleware) addHeaderValue(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handler.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])
	value := uint32(params[3])
	valueLen := uint32(params[4])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "add", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)
	v := mustReadString(mod.Memory(), "value", value, valueLen)

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
func (m *middleware) removeHeader(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handler.HeaderKind(params[0])
	name := uint32(params[1])
	nameLen := uint32(params[2])

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	mustHeaderMutable(ctx, "remove", kind)
	n := mustReadString(mod.Memory(), "name", name, nameLen)

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
func (m *middleware) readBody(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	kind := handler.BodyKind(stack[0])
	buf := uint32(stack[1])
	bufLimit := handler.BufLimit(stack[2])

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

	eofLen := readBody(mod, buf, bufLimit, r)

	stack[0] = eofLen
}

// writeBody implements the WebAssembly host function handler.FuncWriteBody.
func (m *middleware) writeBody(ctx context.Context, mod wazeroapi.Module, params []uint64) {
	kind := handler.BodyKind(params[0])
	buf := uint32(params[1])
	bufLen := uint32(params[2])

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

	writeBody(mod, buf, bufLen, w)
}

// getRemoteAddr implements the WebAssembly host function handler.FuncGetRemoteAddr.
func (m *middleware) getRemoteAddr(ctx context.Context, mod wazeroapi.Module, stack []uint64) {
	buf := uint32(stack[0])
	bufLimit := handler.BufLimit(stack[1])

	method := m.host.GetRemoteAddr(ctx)
	methodLen := writeStringIfUnderLimit(mod.Memory(), buf, bufLimit, method)

	stack[0] = uint64(methodLen)
}

func writeBody(mod wazeroapi.Module, buf, bufLen uint32, w io.Writer) {
	// buf_len 0 means to overwrite with nothing
	var b []byte
	if bufLen > 0 {
		b = mustRead(mod.Memory(), "body", buf, bufLen)
	}
	if _, err := w.Write(b); err != nil { // Write errs if it can't write n bytes
		panic(fmt.Errorf("error writing body: %w", err))
	}
}

// getStatusCode implements the WebAssembly host function
// handler.FuncGetStatusCode.
func (m *middleware) getStatusCode(ctx context.Context, results []uint64) {
	statusCode := m.host.GetStatusCode(ctx)

	results[0] = uint64(statusCode)
}

// setStatusCode implements the WebAssembly host function
// handler.FuncSetStatusCode.
func (m *middleware) setStatusCode(ctx context.Context, params []uint64) {
	statusCode := uint32(params[0])

	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "set", "status code")

	m.host.SetStatusCode(ctx, statusCode)
}

func readBody(mod wazeroapi.Module, buf uint32, bufLimit handler.BufLimit, r io.Reader) (eofLen uint64) {
	// buf_limit 0 serves no purpose as implementations won't return EOF on it.
	if bufLimit == 0 {
		panic(fmt.Errorf("buf_limit==0 reading body"))
	}

	// Allocate a buf to write into directly
	b := mustRead(mod.Memory(), "body", buf, bufLimit)

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
	if s = requestStateFromContext(ctx); s.afterNext {
		panic(fmt.Errorf("can't %s %s after next handler", op, kind))
	}
	return
}

func mustBeforeNextOrFeature(ctx context.Context, feature handler.Features, op, kind string) (s *requestState) {
	if s = requestStateFromContext(ctx); !s.afterNext {
		// Assume this is serving a response from the guest.
	} else if s.features.IsEnabled(feature) {
		// Assume the guest is overwriting the response from next.
	} else {
		panic(fmt.Errorf("can't %s %s after next handler unless %s is enabled",
			op, kind, feature))
	}
	return
}

const i32, i64 = wazeroapi.ValueTypeI32, wazeroapi.ValueTypeI64

func (m *middleware) instantiateHost(ctx context.Context) (wazeroapi.Module, error) {
	return m.runtime.NewHostModuleBuilder(handler.HostModule).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(m.enableFeatures), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("features").Export(handler.FuncEnableFeatures).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getConfig), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handler.FuncGetConfig).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(m.logEnabled), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("level").Export(handler.FuncLogEnabled).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.log), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("level", "message", "message_len").Export(handler.FuncLog).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getMethod), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handler.FuncGetMethod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.setMethod), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("method", "method_len").Export(handler.FuncSetMethod).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getURI), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handler.FuncGetURI).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.setURI), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("uri", "uri_len").Export(handler.FuncSetURI).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getProtocolVersion), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handler.FuncGetProtocolVersion).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getHeaderNames), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "buf", "buf_limit").Export(handler.FuncGetHeaderNames).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getHeaderValues), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "name", "name_len", "buf", "buf_limit").Export(handler.FuncGetHeaderValues).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.setHeaderValue), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len", "value", "value").Export(handler.FuncSetHeaderValue).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.addHeaderValue), []wazeroapi.ValueType{i32, i32, i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len", "value", "value").Export(handler.FuncAddHeaderValue).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.removeHeader), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "name", "name_len").Export(handler.FuncRemoveHeader).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.readBody), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{i64}).
		WithParameterNames("kind", "buf", "buf_limit").Export(handler.FuncReadBody).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.writeBody), []wazeroapi.ValueType{i32, i32, i32}, []wazeroapi.ValueType{}).
		WithParameterNames("kind", "body", "body_len").Export(handler.FuncWriteBody).
		NewFunctionBuilder().
		WithGoModuleFunction(wazeroapi.GoModuleFunc(m.getRemoteAddr), []wazeroapi.ValueType{i32, i32}, []wazeroapi.ValueType{i32}).
		WithParameterNames("buf", "buf_limit").Export(handler.FuncGetRemoteAddr).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(m.getStatusCode), []wazeroapi.ValueType{}, []wazeroapi.ValueType{i32}).
		WithParameterNames().Export(handler.FuncGetStatusCode).
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(m.setStatusCode), []wazeroapi.ValueType{i32}, []wazeroapi.ValueType{}).
		WithParameterNames("status_code").Export(handler.FuncSetStatusCode).
		Instantiate(ctx)
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
func mustReadString(mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) string {
	if byteCount == 0 {
		return ""
	}
	return string(mustRead(mem, fieldName, offset, byteCount))
}

var emptyBody = make([]byte, 0)

// mustRead is like api.Memory except that it panics if the offset and byteCount are out of range.
func mustRead(mem wazeroapi.Memory, fieldName string, offset, byteCount uint32) []byte {
	if byteCount == 0 {
		return emptyBody
	}
	buf, ok := mem.Read(offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}
	return buf
}

func writeIfUnderLimit(mem wazeroapi.Memory, offset, limit handler.BufLimit, v []byte) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mem.Write(offset, v)
	return
}

func writeStringIfUnderLimit(mem wazeroapi.Memory, offset, limit handler.BufLimit, v string) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mem.WriteString(offset, v)
	return
}

type imports uint

const (
	importWasiP1 imports = 1 << iota
	importHttpHandler
)

func detectImports(importedFns []wazeroapi.FunctionDefinition) (imports imports) {
	for _, f := range importedFns {
		moduleName, _, _ := f.Import()
		switch moduleName {
		case handler.HostModule:
			imports |= importHttpHandler
		case wasi_snapshot_preview1.ModuleName:
			imports |= importWasiP1
		}
	}
	return
}
