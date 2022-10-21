package handler

// CountLen describes a possible empty sequence of NUL-terminated strings. For
// compatability with WebAssembly Core Specification 1.0, two uint32 values are
// combined into a single uint64 in the following order:
//
//   - count: zero if the sequence is empty, or the count of strings.
//   - len: possibly zero length of the sequence, including NUL-terminators.
//
// If the uint64 result is zero, the sequence is empty. Otherwise, you need to
// split the results like so.
//
//   - count: `uint32(countLen >> 32)`
//   - len: `uint32(countLen)`
//
// # Examples
//
//   - "": 0<<32|0 or simply zero.
//   - "Accept\0": 1<<32|7
//   - "Content-Type\0Content-Length\0": 2<<32|28
type CountLen = uint64

type BodyKind = uint32

const (
	// BodyKindRequest represents an operation on an HTTP request body.
	//
	// # Notes on FuncReadBody
	//
	// FeatureBufferResponse is required to read the request body without
	// consuming it. To enable it, call FuncEnableFeatures before FuncNext.
	// Otherwise, a downstream handler may panic attempting to read a request
	// body already read upstream.
	//
	// # Notes on FuncWriteBody
	//
	// The first call to FuncWriteBody in FuncHandle overwrites any request
	// body.
	BodyKindRequest BodyKind = 0

	// BodyKindResponse represents an operation on an HTTP request body.
	//
	// # Notes on FuncReadBody
	//
	// FeatureBufferResponse is required to read the response body produced by
	// FuncNext. To enable it, call FuncEnableFeatures beforehand. Otherwise,
	// a handler may panic calling FuncReadBody with BodyKindResponse.
	//
	// # Notes on FuncWriteBody
	//
	// The first call to FuncWriteBody in FuncHandle or after FuncNext
	// overwrites any response body.
	BodyKindResponse BodyKind = 1
)

type HeaderKind = uint32

const (
	// HeaderKindRequest represents an operation on HTTP request headers.
	HeaderKindRequest HeaderKind = 0

	// HeaderKindResponse represents an operation on HTTP response headers.
	HeaderKindResponse HeaderKind = 1

	// HeaderKindRequestTrailers represents an operation on HTTP request
	// trailers (trailing headers). This requires FeatureTrailers.
	//
	// To enable FeatureTrailers, call FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, may result in a panic.
	HeaderKindRequestTrailers HeaderKind = 2

	// HeaderKindResponseTrailers represents an operation on HTTP response
	// trailers (trailing headers). This requires FeatureTrailers.
	//
	// To enable FeatureTrailers, call FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, may result in a panic.
	HeaderKindResponseTrailers HeaderKind = 3
)

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

	// FuncLogEnabled returns 1 if the api.LogLevel is enabled. This value may
	// be cached at request granularity.
	//
	// This function is used to avoid unnecessary overhead generating log
	// messages that the host would discard due to its level being below this.
	//
	// TODO: document on http-wasm-abi
	FuncLogEnabled = "log_enabled"

	// FuncLog logs a message to the host's logs at the given api.LogLevel.
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

	// FuncGetHeaderNames writes all header names for the given HeaderKind,
	// NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is regardless of whether memory was written.
	//
	// TODO: document on http-wasm-abi
	FuncGetHeaderNames = "get_header_names"

	// FuncGetHeaderValues writes all header names of the given HeaderKind and
	// name, NUL-terminated, to memory if the encoded length isn't larger than
	// `buf-limit`. The result is regardless of whether memory was written.
	//
	// TODO: document on http-wasm-abi
	FuncGetHeaderValues = "get_header_values"

	// FuncSetHeaderValue overwrites a header of the given HeaderKind and name
	// with a single value.
	//
	// TODO: document on http-wasm-abi
	FuncSetHeaderValue = "set_header_value"

	// FuncAddHeaderValue adds a single value for the given HeaderKind and
	// name.
	//
	// TODO: document on http-wasm-abi
	FuncAddHeaderValue = "add_header_value"

	// FuncRemoveHeader removes any values for the given HeaderKind and name.
	//
	// TODO: document on http-wasm-abi
	FuncRemoveHeader = "remove_header"

	// FuncReadBody reads up to `buf_limit` bytes remaining in the BodyKind
	// body into memory at offset `buf`. A zero `buf_limit` will panic.
	//
	// The result is `0 or EOF(1) << 32|len`, where `len` is the length in bytes
	// read.
	//
	// `EOF` means the body is exhausted, and future calls return `1<<32|0`
	// (4294967296). `EOF` is not an error, so process `len` bytes returned
	// regardless of `EOF`.
	//
	// Unlike `get_XXX` functions, this function is stateful, so repeated calls
	// reads what's remaining in the stream, as opposed to starting from zero.
	// Callers do not have to exhaust the stream until `EOF`.
	//
	// TODO: document on http-wasm-abi
	FuncReadBody = "read_body"

	// FuncWriteBody reads `buf_len` bytes at memory offset `buf` and writes
	// them to the pending BodyKind body.
	//
	// Unlike `set_XXX` functions, this function is stateful, so repeated calls
	// write to the current stream.
	//
	// TODO: document on http-wasm-abi
	FuncWriteBody = "write_body"

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
)
