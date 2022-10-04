package handler

import (
	"context"

	"github.com/http-wasm/http-wasm-host-go/api"
)

// Middleware is a factory of handler instances implemented in Wasm.
type Middleware[H any] interface {
	// NewHandler creates an HTTP server handler implemented by a WebAssembly
	// module. The returned handler will not invoke FuncNext when it calls
	// FuncSendResponse for reasons such as an authorization failure or serving
	// from cache.
	//
	// ## Notes
	//   - Each handler is independent, so they don't share memory.
	//   - Handlers returned are not safe for concurrent use.
	NewHandler(ctx context.Context, next H) H

	api.Closer
}

// Host implements the host side of the WebAssembly module named HostModule.
// These callbacks are used by the guest function export FuncHandle.
type Host interface {
	// GetPath implements the WebAssembly function export FuncGetPath.
	GetPath(ctx context.Context) string

	// SetPath implements the WebAssembly function export FuncSetPath.
	SetPath(ctx context.Context, path string)

	// GetRequestHeader implements the WebAssembly function export
	// FuncGetRequestHeader. This returns false if the value doesn't exist.
	GetRequestHeader(ctx context.Context, name string) (string, bool)

	// SetResponseHeader implements the WebAssembly function export
	// FuncSetResponseHeader.
	SetResponseHeader(ctx context.Context, name, value string)

	// Next implements the WebAssembly function export FuncNext, which invokes
	// the next handler.
	Next(ctx context.Context)

	// SendResponse implements the WebAssembly function export FuncSendResponse
	// which sends the current response with the given status code and optional
	// body.
	SendResponse(ctx context.Context, statusCode uint32, body []byte)
}
