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
// These callbacks are used by the guest function export FuncHandleRequest.
type Host interface {
	// EnableFeatures supports the WebAssembly function export EnableFeatures.
	EnableFeatures(ctx context.Context, features Features) Features

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
	// FuncGetHeaderNames with HeaderKindRequest. This returns nil if no
	// headers exist.
	GetRequestHeaderNames(ctx context.Context) []string

	// GetRequestHeaderValues supports the WebAssembly function export
	// FuncGetHeaderValues with HeaderKindRequest. This returns nil if no
	// values exist.
	GetRequestHeaderValues(ctx context.Context, name string) []string

	// SetRequestHeaderValue supports the WebAssembly function export
	// FuncSetHeaderValue with HeaderKindRequest.
	SetRequestHeaderValue(ctx context.Context, name, value string)

	// AddRequestHeaderValue supports the WebAssembly function export
	// FuncAddHeaderValue with HeaderKindRequest.
	AddRequestHeaderValue(ctx context.Context, name, value string)

	// RemoveRequestHeader supports the WebAssembly function export
	// FuncRemoveHeader with HeaderKindRequest.
	RemoveRequestHeader(ctx context.Context, name string)

	// RequestBodyReader supports the WebAssembly function export
	// FuncReadBody with BodyKindRequest.
	RequestBodyReader(ctx context.Context) io.ReadCloser

	// RequestBodyWriter supports the WebAssembly function export
	// FuncWriteBody with BodyKindRequest.
	RequestBodyWriter(ctx context.Context) io.Writer

	// GetRequestTrailerNames supports the WebAssembly function export
	// FuncGetHeaderNames with HeaderKindRequestTrailers. This returns nil if
	// no trailers exist or FeatureTrailers is not supported.
	GetRequestTrailerNames(ctx context.Context) []string

	// GetRequestTrailerValues supports the WebAssembly function export
	// FuncGetHeaderValues with HeaderKindRequestTrailers. This returns nil if
	// no values exist or FeatureTrailers is not supported.
	GetRequestTrailerValues(ctx context.Context, name string) []string

	// SetRequestTrailerValue supports the WebAssembly function export
	// FuncSetHeaderValue with HeaderKindRequestTrailers. This panics if
	// FeatureTrailers is not supported.
	SetRequestTrailerValue(ctx context.Context, name, value string)

	// AddRequestTrailerValue supports the WebAssembly function export
	// FuncAddHeaderValue with HeaderKindRequestTrailers. This panics if
	// FeatureTrailers is not supported.
	AddRequestTrailerValue(ctx context.Context, name, value string)

	// RemoveRequestTrailer supports the WebAssembly function export
	// FuncRemoveHeader with HeaderKindRequestTrailers. This panics if
	// FeatureTrailers is not supported.
	RemoveRequestTrailer(ctx context.Context, name string)

	// GetStatusCode supports the WebAssembly function export
	// FuncGetStatusCode.
	GetStatusCode(ctx context.Context) uint32

	// SetStatusCode supports the WebAssembly function export
	// FuncSetStatusCode.
	SetStatusCode(ctx context.Context, statusCode uint32)

	// GetResponseHeaderNames supports the WebAssembly function export
	// FuncGetHeaderNames with HeaderKindResponse. This returns nil if no
	// headers exist.
	GetResponseHeaderNames(ctx context.Context) []string

	// GetResponseHeaderValues supports the WebAssembly function export
	// FuncGetHeaderValues with HeaderKindResponse. This returns nil if no
	// values exist.
	GetResponseHeaderValues(ctx context.Context, name string) []string

	// SetResponseHeaderValue supports the WebAssembly function export
	// FuncSetHeaderValue with HeaderKindResponse.
	SetResponseHeaderValue(ctx context.Context, name, value string)

	// AddResponseHeaderValue supports the WebAssembly function export
	// FuncAddHeaderValue with HeaderKindResponse.
	AddResponseHeaderValue(ctx context.Context, name, value string)

	// RemoveResponseHeader supports the WebAssembly function export
	// FuncRemoveHeader with HeaderKindResponse.
	RemoveResponseHeader(ctx context.Context, name string)

	// ResponseBodyReader supports the WebAssembly function export
	// FuncReadBody with BodyKindResponse.
	ResponseBodyReader(ctx context.Context) io.ReadCloser

	// ResponseBodyWriter supports the WebAssembly function export
	// FuncWriteBody with BodyKindResponse.
	ResponseBodyWriter(ctx context.Context) io.Writer

	// GetResponseTrailerNames supports the WebAssembly function export
	// FuncGetHeaderNames with HeaderKindResponseTrailers. This returns nil if
	// no trailers exist or FeatureTrailers is not supported.
	GetResponseTrailerNames(ctx context.Context) []string

	// GetResponseTrailerValues supports the WebAssembly function export
	// FuncGetHeaderValues with HeaderKindResponseTrailers. This returns nil if
	// no values exist or FeatureTrailers is not supported.
	GetResponseTrailerValues(ctx context.Context, name string) []string

	// SetResponseTrailerValue supports the WebAssembly function export
	// FuncSetHeaderValue with HeaderKindResponseTrailers. This panics if
	// FeatureTrailers is not supported.
	SetResponseTrailerValue(ctx context.Context, name, value string)

	// AddResponseTrailerValue supports the WebAssembly function export
	// FuncAddHeaderValue with HeaderKindResponseTrailers. This panics if
	// FeatureTrailers is not supported.
	AddResponseTrailerValue(ctx context.Context, name, value string)

	// RemoveResponseTrailer supports the WebAssembly function export
	// FuncRemoveHeader with HeaderKindResponseTrailers. This panics if
	// FeatureTrailers is not supported.
	RemoveResponseTrailer(ctx context.Context, name string)
}
