package handler

import (
	"context"
	"io"

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

// Host supports the host side of the WebAssembly module named HostModule.
// These callbacks are used by the guest function export FuncHandle.
type Host interface {
	// EnableFeatures supports the WebAssembly function export EnableFeatures.
	EnableFeatures(ctx context.Context, features Features)

	// GetMethod supports the WebAssembly function export FuncGetMethod.
	GetMethod(ctx context.Context) string

	// SetMethod supports the WebAssembly function export FuncSetMethod.
	SetMethod(ctx context.Context, method string)

	// GetURI supports the WebAssembly function export FuncGetURI.
	GetURI(ctx context.Context) string

	// SetURI supports the WebAssembly function export FuncSetURI.
	SetURI(ctx context.Context, uri string)

	// GetProtocolVersion supports the WebAssembly function export
	// FuncGetProtocolVersion.
	GetProtocolVersion(ctx context.Context) string

	// GetRequestHeaderNames supports the WebAssembly function export
	// FuncGetRequestHeaderNames.
	GetRequestHeaderNames(ctx context.Context) []string

	// GetRequestHeader supports the WebAssembly function export
	// FuncGetRequestHeader. This returns false if the value doesn't exist.
	GetRequestHeader(ctx context.Context, name string) (string, bool)

	// SetRequestHeader supports the WebAssembly function export
	// FuncSetRequestHeader.
	SetRequestHeader(ctx context.Context, name, value string)

	// RequestBodyReader supports the WebAssembly function export
	// FuncReadRequestBody.
	RequestBodyReader(ctx context.Context) io.ReadCloser

	// RequestBodyWriter supports the WebAssembly function export
	// FuncWriteRequestBody.
	RequestBodyWriter(ctx context.Context) io.Writer

	// Next supports the WebAssembly function export FuncNext, which invokes
	// the next handler.
	Next(ctx context.Context)

	// GetStatusCode supports the WebAssembly function export
	// FuncGetStatusCode.
	GetStatusCode(ctx context.Context) uint32

	// SetStatusCode supports the WebAssembly function export
	// FuncSetStatusCode.
	SetStatusCode(ctx context.Context, statusCode uint32)

	// GetResponseHeaderNames supports the WebAssembly function export
	// FuncGetResponseHeaderNames.
	GetResponseHeaderNames(ctx context.Context) []string

	// GetResponseHeader supports the WebAssembly function export
	// FuncGetResponseHeader. This returns false if the value doesn't exist.
	GetResponseHeader(ctx context.Context, name string) (string, bool)

	// SetResponseHeader supports the WebAssembly function export
	// FuncSetResponseHeader.
	SetResponseHeader(ctx context.Context, name, value string)

	// ResponseBodyReader supports the WebAssembly function export
	// FuncReadResponseBody.
	ResponseBodyReader(ctx context.Context) io.ReadCloser

	// ResponseBodyWriter supports the WebAssembly function export
	// FuncWriteResponseBody.
	ResponseBodyWriter(ctx context.Context) io.Writer
}
