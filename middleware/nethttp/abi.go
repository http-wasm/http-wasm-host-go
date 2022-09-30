package wasm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
)

func compileHost(ctx context.Context, r wazero.Runtime, logger httpwasm.Logger) (wazero.CompiledModule, error) {
	h := &host{logger: logger}
	if compiled, err := r.NewHostModuleBuilder("http").
		ExportFunction("log", h.log,
			"log", "ptr", "size").
		ExportFunction("read_request_header", h.readRequestHeader,
			"read_request_header", "name_ptr", "name_size", "value_ptr", "value_limit").
		ExportFunction("set_status_code", h.setStatusCode,
			"set_status_code", "status_code").
		ExportFunction("next", h.next,
			"next").
		Compile(ctx); err != nil {
		return nil, fmt.Errorf("wasm: error compiling host: %w", err)
	} else {
		return compiled, nil
	}
}

func compileGuest(ctx context.Context, r wazero.Runtime, wasm []byte) (wazero.CompiledModule, error) {
	if guest, err := r.CompileModule(ctx, wasm); err != nil {
		return nil, fmt.Errorf("wasm: error compiling guest: %w", err)
	} else if handle, ok := guest.ExportedFunctions()[FuncHandle]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export func[%s]", FuncHandle)
	} else if len(handle.ParamTypes()) != 0 || len(handle.ResultTypes()) != 0 {
		return nil, fmt.Errorf("wasm: guest exports the wrong signature for func[%s]. should be nullary", FuncHandle)
	} else if _, ok = guest.ExportedMemories()[Memory]; !ok {
		return nil, fmt.Errorf("wasm: guest doesn't export memory[%s]", Memory)
	} else {
		return guest, nil
	}
}

// requestStateKey is a context.Context Value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

type requestState struct {
	request    *http.Request
	response   http.ResponseWriter
	handleNext func()
}

func withRequestState(ctx context.Context, response http.ResponseWriter, request *http.Request, next http.Handler) context.Context {
	return context.WithValue(ctx, requestStateKey{}, &requestState{
		request:    request,
		response:   response,
		handleNext: func() { next.ServeHTTP(response, request) },
	})
}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

type host struct {
	logger httpwasm.Logger
}

// log is the WebAssembly function export "http.log", which logs a string.
func (h *host) log(ctx context.Context, m api.Module, ptr, size uint32) {
	msg := requireReadString(ctx, m.Memory(), "msg", ptr, size)
	h.logger(msg)
}

// readRequestHeader is the WebAssembly function export
// "http.read_request_header", which writes a header value to memory, and
// returns the count of bytes written.
//
// # Notes
//
//   - valueLimit is the limit of bytes to write, which means this can
//     truncate unless it is the correct size.
//   - a zero result is possible when the value is empty or the header doesn't
//     exist.
func (h *host) readRequestHeader(ctx context.Context, m api.Module,
	namePtr, nameSize, valuePtr, valueLimit uint32) uint32 {
	// TODO: maybe getRequestHeaderValueSize for to avoid pre-allocating or
	// truncating, and also to tell the difference between no header and empty.
	name := requireReadString(ctx, m.Memory(), "name", namePtr, nameSize)
	r := requestStateFromContext(ctx).request
	if values := r.Header.Values(name); len(values) == 0 {
		return 0
	} else {
		value := values[0]
		size := uint32(len(value))
		if size > valueLimit {
			size = valueLimit
			value = value[0:valueLimit]
		}
		m.Memory().Write(ctx, valuePtr, []byte(value))
		return size
	}
}

// setStatusCode is the WebAssembly function export "http.set_status_code",
// which overwrites the status code of the current response.
func (h *host) setStatusCode(ctx context.Context, statusCode uint32) {
	r := requestStateFromContext(ctx).response
	r.WriteHeader(int(statusCode))
}

// next is the WebAssembly function export "http.next", which invokes the next
// handler. This relies on context state as the real handler isn't known until
// Middleware.NewHandler is invoked.
func (h *host) next(ctx context.Context) {
	requestStateFromContext(ctx).handleNext()
}

// requireReadString is a convenience function that casts requireRead
func requireReadString(ctx context.Context, mem api.Memory, fieldName string, offset, byteCount uint32) string {
	return string(requireRead(ctx, mem, fieldName, offset, byteCount))
}

// requireRead is like api.Memory except that it panics if the offset and byteCount are out of range.
func requireRead(ctx context.Context, mem api.Memory, fieldName string, offset, byteCount uint32) []byte {
	buf, ok := mem.Read(ctx, offset, byteCount)
	if !ok {
		panic(fmt.Errorf("out of memory reading %s", fieldName))
	}
	return buf
}
