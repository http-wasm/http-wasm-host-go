package handler

import (
	"context"

	"github.com/http-wasm/http-wasm-host-go/api"
)

// Middleware is a factory of handler instances implemented in Wasm.
type Middleware[H any] interface {
	// NewHandler creates an HTTP server handler implemented by a WebAssembly
	// module. The returned handler will not invoke FuncNext when it constructs
	// a response in guest wasm for reasons such as an authorization failure or
	// serving from cache.
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
	// EnableFeatures implements the WebAssembly function export EnableFeatures.
	EnableFeatures(ctx context.Context, features Features) Features

	// GetURI implements the WebAssembly function export FuncGetURI.
	GetURI(ctx context.Context) string

	// SetURI implements the WebAssembly function export FuncSetURI.
	SetURI(ctx context.Context, path string)

	// GetRequestHeader implements the WebAssembly function export
	// FuncGetRequestHeader. This returns false if the value doesn't exist.
	GetRequestHeader(ctx context.Context, name string) (string, bool)

	// GetRequestBody implements the WebAssembly function export
	// FuncGetRequestBody.
	GetRequestBody(ctx context.Context) []byte

	// SetRequestBody implements the WebAssembly function export
	// FuncSetRequestBody.
	SetRequestBody(ctx context.Context, body []byte)

	// Next implements the WebAssembly function export FuncNext, which invokes
	// the next handler.
	Next(ctx context.Context)

	// GetStatusCode implements the WebAssembly function export
	// FuncGetStatusCode.
	GetStatusCode(ctx context.Context) uint32

	// SetStatusCode implements the WebAssembly function export
	// FuncSetStatusCode.
	SetStatusCode(ctx context.Context, statusCode uint32)

	// SetResponseHeader implements the WebAssembly function export
	// FuncSetResponseHeader.
	SetResponseHeader(ctx context.Context, name, value string)

	// GetResponseBody implements the WebAssembly function export
	// FuncGetResponseBody.
	GetResponseBody(ctx context.Context) []byte

	// SetResponseBody implements the WebAssembly function export
	// FuncSetResponseBody.
	SetResponseBody(ctx context.Context, body []byte)
}
