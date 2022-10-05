package handler

import (
	"context"
	"fmt"

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
	config                  wazero.ModuleConfig
	logFn                   api.LogFunc
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

	r := &Runtime{host: host, runtime: wr, newNamespace: o.NewNamespace, config: o.ModuleConfig, logFn: o.Logger}

	if r.hostModule, err = r.compileHost(ctx); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	if r.guestModule, err = r.compileGuest(ctx, guest); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}

	return r, nil
}

// Close implements api.Closer
func (r *Runtime) Close(ctx context.Context) error {
	// We don't have to close any guests as the runtime will close it.
	return r.runtime.Close(ctx)
}

type Guest struct {
	ns     wazero.Namespace
	guest  wazeroapi.Module
	handle wazeroapi.Function
}

func (r *Runtime) NewGuest(ctx context.Context) (*Guest, error) {
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

	guest, err := ns.InstantiateModule(ctx, r.guestModule, r.config)
	if err != nil {
		_ = ns.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return &Guest{ns: ns, guest: guest, handle: guest.ExportedFunction(handler.FuncHandle)}, nil
}

// Handle calls the WebAssembly function export "handle".
func (g *Guest) Handle(ctx context.Context) (err error) {
	_, err = g.handle.Call(ctx)
	return
}

// Close implements api.Closer
func (g *Guest) Close(ctx context.Context) error {
	// Closing the namespace closes both the host and guest modules
	return g.ns.Close(ctx)
}

// getPath is the WebAssembly function export named handler.FuncGetPath which
// writes the request path value to memory, if it isn't larger than the buffer
// size limit. The result is the actual path length in bytes.
func (r *Runtime) getPath(ctx context.Context, mod wazeroapi.Module,
	buf, bufLimit uint32) (result uint32) {
	path := r.host.GetPath(ctx)
	result = uint32(len(path))
	if result > bufLimit {
		return // caller can retry with a larger bufLimit
	}
	mod.Memory().WriteString(ctx, buf, path)
	return
}

// setPath is the WebAssembly function export named handler.FuncSetPath which
// overwrites the request path with one read from memory.
func (r *Runtime) setPath(ctx context.Context, mod wazeroapi.Module,
	path, pathLen uint32) {
	p := mustReadString(ctx, mod.Memory(), "path", path, pathLen)
	r.host.SetPath(ctx, p)
}

// getRequestHeader is the WebAssembly function export named
// handler.FuncGetRequestHeader which writes a header value to memory if it
// exists and isn't larger than the buffer size limit. The result is
// `1<<32|value_len` or zero if the header doesn't exist.
func (r *Runtime) getRequestHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, value, valueLimit uint32) (result uint64) {
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v, ok := r.host.GetRequestHeader(ctx, n)
	if !ok {
		return // value doesn't exist
	}
	length := uint32(len(v))
	result = uint64(1<<32) | uint64(length)
	if length > valueLimit {
		return // caller can retry with a larger bufLimit
	}
	mod.Memory().WriteString(ctx, value, v)
	return
}

// setResponseHeader is the WebAssembly function export named
// handler.FuncSetResponseHeader which sets a response header from a name and
// value read from memory.
func (r *Runtime) setResponseHeader(ctx context.Context, mod wazeroapi.Module,
	name, nameLen, value, valueLen uint32) {
	n := mustReadString(ctx, mod.Memory(), "name", name, nameLen)
	v := mustReadString(ctx, mod.Memory(), "value", value, valueLen)
	r.host.SetResponseHeader(ctx, n, v)
}

// sendResponse is the WebAssembly function export named
// handler.FuncSendResponse which sends the HTTP response with a given status
// code and optional body.
func (r *Runtime) sendResponse(ctx context.Context, mod wazeroapi.Module,
	statusCode, body, bodyLen uint32) {
	b := mustRead(ctx, mod.Memory(), "body", body, bodyLen)
	r.host.SendResponse(ctx, statusCode, b)
}

func (r *Runtime) compileHost(ctx context.Context) (wazero.CompiledModule, error) {
	if compiled, err := r.runtime.NewHostModuleBuilder(handler.HostModule).
		ExportFunction(handler.FuncLog, r.log,
			handler.FuncLog, "message", "message_len").
		ExportFunction(handler.FuncGetPath, r.getPath,
			handler.FuncGetPath, "buf", "buf_limit").
		ExportFunction(handler.FuncSetPath, r.setPath,
			handler.FuncSetPath, "path", "path_len").
		ExportFunction(handler.FuncGetRequestHeader, r.getRequestHeader,
			handler.FuncGetRequestHeader, "name", "name_len", "buf", "buf_limit").
		ExportFunction(handler.FuncSetResponseHeader, r.setResponseHeader,
			handler.FuncSetResponseHeader, "name", "name_len", "value", "value_len").
		ExportFunction(handler.FuncSendResponse, r.sendResponse,
			handler.FuncSendResponse, "status_code", "body", "body_len").
		ExportFunction(handler.FuncNext, r.host.Next,
			handler.FuncNext).
		Compile(ctx); err != nil {
		return nil, fmt.Errorf("wasm: error compiling host: %w", err)
	} else {
		return compiled, nil
	}
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

// log implements the WebAssembly function export "log". It has
// the same signature as api.LogFunc.
func (r *Runtime) log(ctx context.Context, mod wazeroapi.Module,
	message, messageLen uint32) {
	m := mustReadString(ctx, mod.Memory(), "message", message, messageLen)
	r.logFn(ctx, m)
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
