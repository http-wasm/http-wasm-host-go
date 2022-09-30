package wasm

import (
	"context"
	"net/http"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	FuncHandle = "handle"
	Memory     = "memory"
)

// Handler is a http.Handler implemented by a Wasm module. `ServeHTTP` is
// dispatched to FuncHandle, which is exported by the guest.
type Handler interface {
	http.Handler

	// Close releases the Wasm module associated with this handler.
	Close(context.Context) error
}

// compile-time check to ensure wasmHandler implements Handler.
var _ Handler = &wasmHandler{}

type wasmHandler struct {
	ns    wazero.Namespace
	guest api.Module
	next  http.Handler
}

// ServeHTTP implements http.Handler
func (w *wasmHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	ctx := withRequestState(request.Context(), response, request, w.next)
	if _, err := w.guest.ExportedFunction("handle").Call(ctx); err != nil {
		// TODO: after testing, shouldn't send errors into the HTTP response.
		response.Write([]byte(err.Error())) // nolint
		response.WriteHeader(500)
	}
}

// Close implements Handler.Close
func (w *wasmHandler) Close(ctx context.Context) error {
	// We don't have to close the guest as the namespace will close it.
	return w.ns.Close(ctx)
}
