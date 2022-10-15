// Package internalhandler is not named handler as doing so interferes with
// godoc links for the api handler package.
package internalhandler

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	wazeroapi "github.com/tetratelabs/wazero/api"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/internal"
)

type Runtime struct {
	host                    handler.Host
	runtime                 wazero.Runtime
	hostModule, guestModule wazero.CompiledModule
	newNamespace            httpwasm.NewNamespace
	moduleConfig            wazero.ModuleConfig
	guestConfig             []byte
	logFn                   api.LogFunc
	pool                    sync.Pool
	Features                handler.Features
}

func NewRuntime(ctx context.Context, guest []byte, host handler.Host, options ...httpwasm.Option) (*Runtime, error) {
	o := &internal.WazeroOptions{
		NewRuntime:   internal.DefaultRuntime,
		NewNamespace: internal.DefaultNamespace,
		ModuleConfig: wazero.NewModuleConfig(),
		Logger:       func(context.Context, string) {},
	}
	for _, option := range options {
		option(o)
	}

	wr, err := o.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating runtime: %w", err)
	}

	r := &Runtime{
		host:         host,
		runtime:      wr,
		newNamespace: o.NewNamespace,
		moduleConfig: o.ModuleConfig,
		guestConfig:  o.GuestConfig,
		logFn:        o.Logger,
	}

	if r.hostModule, err = r.compileHost(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	if r.guestModule, err = r.compileGuest(ctx, guest); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	if g, err := r.newGuest(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, err
	} else {
		r.pool.Put(g)
	}

	return r, nil
}

func (r *Runtime) compileGuest(ctx context.Context, wasm []byte) (wazero.CompiledModule, error) {
	if guest, err := r.runtime.CompileModule(ctx, wasm); err != nil {
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

// Handle handles a request by calling guest.handle.
func (r *Runtime) Handle(ctx context.Context) error {
	poolG := r.pool.Get()
	if poolG == nil {
		g, err := r.newGuest(ctx)
		if err != nil {
			return err
		}
		poolG = g
	}
	g := poolG.(*guest)
	defer r.pool.Put(g)
	s := &requestState{features: r.Features}
	defer s.Close()
	ctx = context.WithValue(ctx, requestStateKey{}, s)
	return g.handle(ctx)
}

// next calls the same function as documented on handler.Host.
func (r *Runtime) next(ctx context.Context) {
	s := requestStateFromContext(ctx)
	if s.calledNext {
		panic("already called next")
	}
	s.calledNext = true
	_ = s.closeRequest()
	r.host.Next(ctx)
}

// Close implements api.Closer
func (r *Runtime) Close(ctx context.Context) error {
	// We don't have to close any guests as the runtime will close it.
	return r.runtime.Close(ctx)
}

type guest struct {
	ns         wazero.Namespace
	guest      wazeroapi.Module
	handleFunc wazeroapi.Function
}

func (r *Runtime) newGuest(ctx context.Context) (*guest, error) {
	ns, err := r.newNamespace(ctx, r.runtime)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating namespace: %w", err)
	}

	// Note: host modules don't use configuration
	_, err = ns.InstantiateModule(ctx, r.hostModule, wazero.NewModuleConfig())
	if err != nil {
		_ = ns.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating host: %w", err)
	}

	g, err := ns.InstantiateModule(ctx, r.guestModule, r.moduleConfig)
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
func (r *Runtime) enableFeatures(ctx context.Context, features uint64) uint64 {
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.features = s.features.WithEnabled(handler.Features(features))
		r.host.EnableFeatures(ctx, s.features)
		return uint64(s.features)
	} else {
		r.Features = r.Features.WithEnabled(handler.Features(features))
		return uint64(r.Features)
	}
}

// getConfig implements the WebAssembly host function handler.FuncGetConfig.
func (r *Runtime) getConfig(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (len uint32) {
	return writeIfUnderLimit(ctx, mod, buf, bufLimit, r.guestConfig)
}

// log implements the WebAssembly host function handler.FuncLog.
func (r *Runtime) log(ctx context.Context, mod wazeroapi.Module,
	message, messageLen uint32) {
	var m string
	if messageLen > 0 {
		m = mustReadString(ctx, mod.Memory(), "message", message, messageLen)
	}
	r.logFn(ctx, m)
}

// getMethod implements the WebAssembly host function handler.FuncGetMethod.
func (r *Runtime) getMethod(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (len uint32) {
	method := r.host.GetMethod(ctx)
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, method)
}

// getRequestHeader implements the WebAssembly host function
// handler.FuncSetMethod.
func (r *Runtime) setMethod(ctx context.Context, mod wazeroapi.Module,
	method, methodLen uint32) {
	_ = mustBeforeNext(ctx, "set method")

	var p string
	if methodLen == 0 {
		panic("HTTP method cannot be empty")
	}
	p = mustReadString(ctx, mod.Memory(), "method", method, methodLen)
	r.host.SetMethod(ctx, p)
}

// getURI implements the WebAssembly host function handler.FuncGetURI.
func (r *Runtime) getURI(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (len uint32) {
	uri := r.host.GetURI(ctx)
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, uri)
}

// getRequestHeader implements the WebAssembly host function
// handler.FuncSetURI.
func (r *Runtime) setURI(ctx context.Context, mod wazeroapi.Module,
	uri, uriLen uint32) {
	var p string
	if uriLen > 0 { // overwrite with empty is supported
		p = mustReadString(ctx, mod.Memory(), "uri", uri, uriLen)
	}
	r.host.SetURI(ctx, p)
}

// getProtocolVersion implements the WebAssembly host function
// handler.FuncGetProtocolVersion.
func (r *Runtime) getProtocolVersion(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) uint32 {
	protocolVersion := r.host.GetProtocolVersion(ctx)
	if len(protocolVersion) == 0 {
		panic("HTTP protocol version cannot be empty")
	}
	return writeStringIfUnderLimit(ctx, mod, buf, bufLimit, protocolVersion)
}

// getRequestHeaderNames implements the WebAssembly host function
// handler.FuncGetRequestHeaderNames.
func (r *Runtime) getRequestHeaderNames(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (len uint32) {
	headers := r.host.GetRequestHeaderNames(ctx)
	return writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, headers)
}

// getRequestHeader implements the WebAssembly host function
// handler.FuncGetRequestHeader.
func (r *Runtime) getRequestHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, buf, bufLimit uint32) (result uint64) {
	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v, ok := r.host.GetRequestHeader(ctx, n)
	if !ok {
		return // value doesn't exist
	}
	result = uint64(1<<32) | uint64(writeStringIfUnderLimit(ctx, mod, buf, bufLimit, v))
	return
}

// setRequestHeader implements the WebAssembly host function
// handler.FuncRequestHeader.
func (r *Runtime) setRequestHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, value, valueLen uint32) {
	_ = mustBeforeNext(ctx, "set request header")

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v := mustReadString(ctx, mod.Memory(), "value", value, valueLen)
	r.host.SetRequestHeader(ctx, n, v)
}

// readRequestBody implements the WebAssembly host function
// handler.FuncReadRequestBody.
func (r *Runtime) readRequestBody(ctx context.Context, mod wazeroapi.Module,
	buf, bufLen uint32) (result uint64) {
	s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "read response body")

	// Lazy create the reader.
	reader := s.requestBodyReader
	if reader == nil {
		reader = r.host.RequestBodyReader(ctx)
		s.requestBodyReader = reader
	}

	return readBody(ctx, mod, buf, bufLen, reader)
}

// writeRequestBody implements the WebAssembly host function
// handler.FuncWriteRequestBody.
func (r *Runtime) writeRequestBody(ctx context.Context, mod wazeroapi.Module,
	buf, bufLen uint32) {
	s := mustBeforeNext(ctx, "write request body")

	// Lazy create the writer.
	w := s.requestBodyWriter
	if w == nil {
		w = r.host.RequestBodyWriter(ctx)
		s.requestBodyWriter = w
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
func (r *Runtime) getStatusCode(ctx context.Context) uint32 {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "get status code")

	return r.host.GetStatusCode(ctx)
}

// setStatusCode implements the WebAssembly host function
// handler.FuncSetStatusCode.
func (r *Runtime) setStatusCode(ctx context.Context, statusCode uint32) {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "set status code")

	r.host.SetStatusCode(ctx, statusCode)
}

// getResponseHeaderNames implements the WebAssembly host function
// handler.FuncGetResponseHeaderNames.
func (r *Runtime) getResponseHeaderNames(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (len uint32) {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "get response header names")

	headers := r.host.GetResponseHeaderNames(ctx)
	return writeNULTerminated(ctx, mod.Memory(), buf, bufLimit, headers)
}

// getResponseHeader implements the WebAssembly host function
// handler.FuncGetResponseHeader.
func (r *Runtime) getResponseHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, buf, bufLimit uint32) (result uint64) {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "get response header")

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v, ok := r.host.GetResponseHeader(ctx, n)
	if !ok {
		return // value doesn't exist
	}
	result = uint64(1<<32) | uint64(writeStringIfUnderLimit(ctx, mod, buf, bufLimit, v))
	return
}

// setResponseHeader implements the WebAssembly host function
// handler.FuncRequestHeader.
func (r *Runtime) setResponseHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, value, valueLen uint32) {
	_ = mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "set response header")

	if nameLen == 0 {
		panic("HTTP header name cannot be empty")
	}
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v := mustReadString(ctx, mod.Memory(), "value", value, valueLen)
	r.host.SetResponseHeader(ctx, n, v)
}

// readResponseBody implements the WebAssembly host function
// handler.FuncReadResponseBody.
func (r *Runtime) readResponseBody(ctx context.Context, mod wazeroapi.Module,
	buf, bufLen uint32) (result uint64) {
	s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "read response body")

	// Lazy create the reader.
	reader := s.responseBodyReader
	if reader == nil {
		reader = r.host.ResponseBodyReader(ctx)
		s.responseBodyReader = reader
	}

	return readBody(ctx, mod, buf, bufLen, reader)
}

func readBody(ctx context.Context, mod wazeroapi.Module, buf, bufLen uint32, r io.Reader) uint64 {
	// buf_len 0 serves no purpose as implementations won't return EOF on it.
	if bufLen == 0 {
		panic(fmt.Errorf("buf_len==0 reading body"))
	}

	// Allocate a buf to write into directly
	b := mustRead(ctx, mod.Memory(), "body", buf, bufLen)

	// Attempt to fill the buffer until an error occurs. Notably, this works
	// around a full read not returning EOF until the
	var err error
	n := uint32(0)
	for n < bufLen && err == nil {
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

// writeResponseBody implements the WebAssembly host function
// handler.FuncWriteResponseBody.
func (r *Runtime) writeResponseBody(ctx context.Context, mod wazeroapi.Module,
	buf, bufLen uint32) {
	s := mustBeforeNextOrFeature(ctx, handler.FeatureBufferResponse, "write response body")

	// Lazy create the writer.
	w := s.responseBodyWriter
	if w == nil {
		w = r.host.ResponseBodyWriter(ctx)
		s.responseBodyWriter = w
	}

	writeBody(ctx, mod, buf, bufLen, w)
}

func mustBeforeNext(ctx context.Context, op string) (s *requestState) {
	if s = requestStateFromContext(ctx); s.calledNext {
		panic(fmt.Errorf("can't %s response after next handler", op))
	}
	return
}

func mustBeforeNextOrFeature(ctx context.Context, feature handler.Features, op string) (s *requestState) {
	if s = requestStateFromContext(ctx); !s.calledNext {
		// Assume this is serving a response from the guest.
	} else if s.features.IsEnabled(feature) {
		// Assume the guest is overwriting the response from next.
	} else {
		panic(fmt.Errorf("can't %s after next handler unless %s is enabled",
			op, feature))
	}
	return
}

func (r *Runtime) compileHost(ctx context.Context) (wazero.CompiledModule, error) {
	if compiled, err := r.runtime.NewHostModuleBuilder(handler.HostModule).
		ExportFunction(handler.FuncEnableFeatures, r.enableFeatures,
			handler.FuncEnableFeatures, "features").
		ExportFunction(handler.FuncGetConfig, r.getConfig,
			handler.FuncGetConfig, "buf", "buf_limit").
		ExportFunction(handler.FuncLog, r.log,
			handler.FuncLog, "message", "message_len").
		ExportFunction(handler.FuncGetMethod, r.getMethod,
			handler.FuncGetMethod, "buf", "buf_limit").
		ExportFunction(handler.FuncSetMethod, r.setMethod,
			handler.FuncSetMethod, "method", "method_len").
		ExportFunction(handler.FuncGetURI, r.getURI,
			handler.FuncGetURI, "buf", "buf_limit").
		ExportFunction(handler.FuncSetURI, r.setURI,
			handler.FuncSetURI, "uri", "uri_len").
		ExportFunction(handler.FuncGetProtocolVersion, r.getProtocolVersion,
			handler.FuncGetProtocolVersion, "buf", "buf_limit").
		ExportFunction(handler.FuncGetRequestHeaderNames, r.getRequestHeaderNames,
			handler.FuncGetRequestHeaderNames, "buf", "buf_limit").
		ExportFunction(handler.FuncGetRequestHeader, r.getRequestHeader,
			handler.FuncGetRequestHeader, "name", "name_len", "buf", "buf_limit").
		ExportFunction(handler.FuncSetRequestHeader, r.setRequestHeader,
			handler.FuncSetRequestHeader, "name", "name_len", "value", "value_len").
		ExportFunction(handler.FuncReadRequestBody, r.readRequestBody,
			handler.FuncReadRequestBody, "buf", "buf_limit").
		ExportFunction(handler.FuncWriteRequestBody, r.writeRequestBody,
			handler.FuncWriteRequestBody, "body", "body_len").
		ExportFunction(handler.FuncNext, r.next,
			handler.FuncNext).
		ExportFunction(handler.FuncGetStatusCode, r.getStatusCode,
			handler.FuncGetStatusCode).
		ExportFunction(handler.FuncSetStatusCode, r.setStatusCode,
			handler.FuncSetStatusCode, "status_code").
		ExportFunction(handler.FuncGetResponseHeaderNames, r.getResponseHeaderNames,
			handler.FuncGetResponseHeaderNames, "buf", "buf_limit").
		ExportFunction(handler.FuncGetResponseHeader, r.getResponseHeader,
			handler.FuncGetResponseHeader, "name", "name_len", "buf", "buf_limit").
		ExportFunction(handler.FuncSetResponseHeader, r.setResponseHeader,
			handler.FuncSetResponseHeader, "name", "name_len", "value", "value_len").
		ExportFunction(handler.FuncReadResponseBody, r.readResponseBody,
			handler.FuncReadResponseBody, "buf", "buf_len").
		ExportFunction(handler.FuncWriteResponseBody, r.writeResponseBody,
			handler.FuncWriteResponseBody, "body", "body_len").
		Compile(ctx); err != nil {
		return nil, fmt.Errorf("wasm: error compiling host: %w", err)
	} else {
		return compiled, nil
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

func writeIfUnderLimit(ctx context.Context, mod wazeroapi.Module, offset, limit uint32, v []byte) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mod.Memory().Write(ctx, offset, v)
	return
}

func writeStringIfUnderLimit(ctx context.Context, mod wazeroapi.Module, offset, limit uint32, v string) (vLen uint32) {
	vLen = uint32(len(v))
	if vLen > limit {
		return // caller can retry with a larger limit
	} else if vLen == 0 {
		return // nothing to write
	}
	mod.Memory().WriteString(ctx, offset, v)
	return
}
