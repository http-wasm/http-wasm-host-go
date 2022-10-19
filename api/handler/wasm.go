package handler

const (
	// HostModule is the WebAssembly module name of the ABI this middleware
	// implements.
	//
	// Note: This is lower-hyphen case while functions are lower_underscore to
	// follow conventions of wit-bindgen, which retains the case format of the
	// interface filename as the module name, but converts function and
	// parameter names to lower_underscore format.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md
	HostModule = "http-handler"

	// FuncEnableFeatures tries to enable the given features and returns the
	// Features bitflag supported by the host. This must be called prior to
	// FuncNext to enable Features needed by the guest.
	//
	// This may be called prior to FuncHandle, for example inside a start
	// function. Doing so reduces overhead per-call and also allows the guest
	// to fail early on unsupported.
	//
	// If called during FuncHandle, any new features are only enabled for the
	// scope of the current request. This allows fine-grained access to
	// expensive features such as FeatureBufferResponse.
	//
	// TODO: document on http-wasm-abi
	FuncEnableFeatures = "enable_features"

	// FuncGetConfig writes configuration from the host to memory if it exists
	// and isn't larger than `buf-limit`. The result is its length in bytes.
	//
	// Note: Configuration is determined by the guest and is not necessarily
	// UTF-8 encoded.
	//
	// TODO: document on http-wasm-abi
	FuncGetConfig = "get_config"

	// FuncLog logs a message to the host's logs.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#log
	FuncLog = "log"

	// FuncHandle is the entrypoint guest export called by the host when
	// processing a request.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#handle
	FuncHandle = "handle"

	// FuncGetMethod writes the method to memory if it isn't larger than
	// `buf-limit`. The result is its length in bytes. Ex. "GET"
	//
	// TODO: document on http-wasm-abi
	FuncGetMethod = "get_method"

	// FuncSetMethod overwrites the method with one read from memory.
	//
	// TODO: document on http-wasm-abi
	FuncSetMethod = "set_method"

	// FuncGetURI writes the URI to memory if it isn't larger than `buf-limit`.
	// The result is its length in bytes. Ex. "/v1.0/hi?name=panda"
	//
	// Note: The URI may include query parameters.
	//
	// TODO: update document on http-wasm-abi
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-uri
	FuncGetURI = "get_uri"

	// FuncSetURI overwrites the URI with one read from memory.
	//
	// Note: The URI may include query parameters.
	//
	// TODO: update document on http-wasm-abi
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-uri
	FuncSetURI = "set_uri"

	// FuncGetProtocolVersion writes the HTTP protocol version to memory if it
	// isn't larger than `buf-limit`. The result is its length in bytes.
	// Ex. "HTTP/1.1"
	//
	// See https://www.rfc-editor.org/rfc/rfc9110#name-protocol-version
	// TODO: document on http-wasm-abi
	FuncGetProtocolVersion = "get_protocol_version"

	// FuncGetRequestHeaderNames writes all header names, NUL-terminated, to
	// memory if the encoded length isn't larger than `buf-limit`. The result
	// is the encoded length in bytes. Ex. "Accept\0Date\0"
	//
	// TODO: document on http-wasm-abi
	FuncGetRequestHeaderNames = "get_request_header_names"

	// FuncGetRequestHeader writes a header value to memory if it exists and
	// isn't larger than `buf-limit`. The result is `1<<32|len`, where `len` is
	// the bytes written, or zero if the header doesn't exist.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-request-header
	FuncGetRequestHeader = "get_request_header"

	// FuncGetRequestHeaders writes all header values, NUL-terminated, to
	// memory if the encoded length isn't larger than `buf-limit`. The result
	// is the encoded length in bytes. Ex. "val1\0val2\0" == 10
	//
	// TODO: document on http-wasm-abi
	FuncGetRequestHeaders = "get_request_headers"

	// FuncSetRequestHeader overwrites a request header with a given name to
	// a value read from memory.
	//
	// TODO: document on http-wasm-abi
	FuncSetRequestHeader = "set_request_header"

	// FuncAddRequestHeader adds a request header with a given name using a
	// value read from memory.
	//
	// TODO: document on http-wasm-abi
	FuncAddRequestHeader = "add_request_header"

	// FuncRemoveRequestHeader removes any values for the given header name.
	//
	// TODO: document on http-wasm-abi
	FuncRemoveRequestHeader = "remove_request_header"

	// FuncReadRequestBody reads up to `buf_limit` bytes remaining in the body
	// into memory at offset `buf`. A zero `buf_limit` will panic. The result
	// is `0 or EOF(1) << 32|len`, where `len` is the possibly zero length of
	// bytes read.
	//
	// `EOF` means the body is exhausted, and future calls return `1<<32|0`
	// (4294967296). `EOF` is not an error, so process `len` bytes returned
	// regardless of `EOF`.
	//
	// Unlike `get_XXX` functions, this function is stateful, so repeated calls
	// reads what's remaining in the stream, as opposed to starting from zero.
	// Callers do not have to exhaust the stream until `EOF`.
	//
	// To allow downstream handlers to read the original request body, enable
	// FeatureBufferRequest via FuncEnableFeatures. Otherwise, create a
	// response inside the guest, or write an appropriate body via
	// FuncWriteRequestBody before calling FuncNext.
	//
	// TODO: document on http-wasm-abi
	FuncReadRequestBody = "read_request_body"

	// FuncWriteRequestBody reads `buf_len` bytes at memory offset `buf` and
	// writes them to the pending request body. The first call overwrites any
	// request body.
	//
	// Unlike `set_XXX` functions, this function is stateful, so repeated calls
	// write to the current stream.
	//
	// Note: This can only be called before FuncNext.
	// TODO: document on http-wasm-abi
	FuncWriteRequestBody = "write_request_body"

	// FuncGetRequestTrailerNames writes all trailing header (trailer) names,
	// NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is the encoded length in bytes.
	// Ex. "ChecksumSHA256\0"
	//
	// TODO: document on http-wasm-abi
	FuncGetRequestTrailerNames = "get_request_trailer_names"

	// FuncGetRequestTrailer writes a trailing header (trailer) value to memory
	// if it exists and isn't larger than `buf-limit`. The result is
	// `1<<32|len`, where `len` is the bytes written, or zero if the trailer
	// doesn't exist.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-request-trailer
	FuncGetRequestTrailer = "get_request_trailer"

	// FuncGetRequestTrailers writes all trailing header (trailer) values,
	// NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is the encoded length in bytes.
	// Ex. "val1\0val2\0" == 10
	//
	// Note: `len` is the encoded length: "val\0" == 4 or "val1\0val2\0" == 10
	//
	// TODO: document on http-wasm-abi
	FuncGetRequestTrailers = "get_request_trailers"

	// FuncSetRequestTrailer overwrites a request trailer with a given name to
	// a value read from memory.
	//
	// TODO: document on http-wasm-abi
	FuncSetRequestTrailer = "set_request_trailer"

	// FuncAddRequestTrailer adds a trailing header (trailer) with a given name
	// using a value read from memory.
	//
	// TODO: document on http-wasm-abi
	FuncAddRequestTrailer = "add_request_trailer"

	// FuncRemoveRequestTrailer removes any values for the given trailer name.
	//
	// To use this function after FuncNext, set FeatureBufferRequest via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncRemoveRequestTrailer = "remove_request_trailer"

	// FuncNext calls a downstream handler and blocks until it is finished
	// processing.
	//
	// By default, whether the next handler flushes the response prior to
	// returning is implementation-specific. If your handler needs to inspect
	// or manipulate the downstream response, enable FeatureBufferResponse via
	// FuncEnableFeatures prior to calling this function.
	//
	// TODO: update existing document on http-wasm-abi
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#next
	FuncNext = "next"

	// FuncGetStatusCode returns the status code produced by FuncNext. This
	// requires FeatureBufferResponse.
	//
	// To enable FeatureBufferResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetStatusCode = "get_status_code"

	// FuncSetStatusCode overrides the status code. The default is 200.
	//
	// To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncSetStatusCode = "set_status_code"

	// FuncGetResponseHeaderNames writes all header names, NUL-terminated, to
	// memory if the encoded length isn't larger than `buf-limit`. The result
	// is the encoded length in bytes. Ex. "Content-Length\0Content-Type\0"
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseHeaderNames = "get_response_header_names"

	// FuncGetResponseHeader writes a header value to memory if it exists and
	// isn't larger than `buf-limit`. The result is `1<<32|len`, where `len` is
	// the bytes written, or zero if the header doesn't exist. This requires
	// FeatureBufferResponse.
	//
	// To enable FeatureBufferResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseHeader = "get_response_header"

	// FuncGetResponseHeaders writes all header values, NUL-terminated, to
	// memory if the encoded length isn't larger than `buf-limit`. The result
	// is the encoded length in bytes. Ex. "val1\0val2\0" == 10
	//
	// To enable FeatureBufferResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// Note: `len` is the encoded length: "val\0" == 4 or "val1\0val2\0" == 10
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseHeaders = "get_response_headers"

	// FuncSetResponseHeader overwrites a response header with a given name to
	// a value read from memory.
	//
	// To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-response-header
	FuncSetResponseHeader = "set_response_header"

	// FuncAddResponseHeader adds a response header with a given name using a
	// value read from memory.
	//
	// To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncAddResponseHeader = "add_response_header"

	// FuncRemoveResponseHeader removes any values for the given header name.
	//
	// To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncRemoveResponseHeader = "remove_response_header"

	// FuncReadResponseBody reads up to `buf_limit` bytes remaining in the body
	// into memory at offset `buf`. A zero `buf_limit` will panic. The result
	// is `0 or EOF(1) << 32|len`, where `len` is the possibly zero length of
	// bytes read.
	//
	// `EOF` means the body is exhausted, and future calls return `1<<32|0`
	// (4294967296). `EOF` is not an error, so process `len` bytes returned
	// regardless of `EOF`.
	//
	// Unlike `get_XXX` functions, this function is stateful, so repeated calls
	// reads what's remaining in the stream, as opposed to starting from zero.
	// Callers do not have to exhaust the stream until `EOF`.
	//
	// Note: This function requires FeatureBufferResponse. To enable it, call
	// FuncEnableFeatures prior to FuncNext. Doing otherwise, or calling before
	// FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncReadResponseBody = "read_response_body"

	// FuncWriteResponseBody reads `buf_len` bytes at memory offset `buf` and
	// writes them to the pending response body. The first call to this in
	// FuncHandle or after FuncNext overwrites any response body.
	//
	// Unlike `set_XXX` functions, this function is stateful, so repeated calls
	// write to the current stream.
	//
	// Note: To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	// TODO: document on http-wasm-abi
	FuncWriteResponseBody = "write_response_body"

	// FuncGetResponseTrailerNames writes all trailing header (trailer) names,
	// NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is the encoded length in bytes.
	// Ex. "grpc-status\0"
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseTrailerNames = "get_response_trailer_names"

	// FuncGetResponseTrailer writes a trailing header (trailer) value to
	// memory if it exists and isn't larger than `buf-limit`. The result is
	// `1<<32|len`, where `len` is the bytes written, or zero if the trailer
	// doesn't exist. This requires FeatureBufferResponse.
	//
	// To enable FeatureBufferResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseTrailer = "get_response_trailer"

	// FuncGetResponseTrailers writes all trailing header (trailer) values,
	// NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is the encoded length in bytes.
	// Ex. "val1\0val2\0" == 10
	//
	// To enable FeatureBufferResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// Note: `len` is the encoded length: "val\0" == 4 or "val1\0val2\0" == 10
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseTrailers = "get_response_trailers"

	// FuncSetResponseTrailer overwrites a trailing header (trailer) with a
	// given name to a value read from memory.
	//
	// Note: To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncSetResponseTrailer = "set_response_trailer"

	// FuncAddResponseTrailer adds a trailing header (trailer) with a given
	// name using a value read from memory.
	//
	// Note: To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncAddResponseTrailer = "add_response_trailer"

	// FuncRemoveResponseTrailer removes any values for the given trailer name.
	//
	// To use this function after FuncNext, set FeatureBufferResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncRemoveResponseTrailer = "remove_response_trailer"
)
