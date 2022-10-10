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
	// expensive features such as FeatureCaptureResponse.
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

	// FuncGetRequestHeader writes a header value to memory if it exists and
	// isn't larger than `buf-limit`. The result is `1<<32|len`, where `len` is
	// the bytes written, or zero if the header doesn't exist.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-request-header
	FuncGetRequestHeader = "get_request_header"

	// FuncGetURI writes the path to memory if it isn't larger than
	// `buf-limit`. The result is its length in bytes.
	//
	// Note: The path does not include query parameters.
	//
	// TODO: update document on http-wasm-abi
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-path
	FuncGetURI = "get_uri"

	// FuncSetURI Overwrites the request URI with one read from memory.
	//
	// Note: The URI includes any query parameters.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-uri
	FuncSetURI = "set_uri"

	// FuncNext calls a downstream handler and blocks until it is finished
	// processing.
	//
	// By default, whether the next handler flushes the response prior to
	// returning is implementation-specific. If your handler needs to inspect
	// or manipulate the downstream response, enable FeatureCaptureResponse via
	// FuncEnableFeatures prior to calling this function.
	//
	// TODO: update existing document on http-wasm-abi
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#next
	FuncNext = "next"

	// FuncGetResponseHeader writes a header value to memory if it exists and
	// isn't larger than `buf-limit`. The result is `1<<32|len`, where `len` is
	// the bytes written, or zero if the header doesn't exist. This requires
	// FeatureCaptureResponse.
	//
	// To enable FeatureCaptureResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseHeader = "get_response_header"

	// FuncSetResponseHeader overwrites a response header with a given name to
	// a value read from memory.
	//
	// To use this function after FuncNext, set FeatureCaptureResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-response-header
	FuncSetResponseHeader = "set_response_header"

	// FuncGetStatusCode gets the status code produced by FuncNext. This
	// requires FeatureCaptureResponse.
	//
	// To enable FeatureCaptureResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetStatusCode = "get_status_code"

	// FuncSetStatusCode overrides the status code. The default is 200.
	//
	// To use this function after FuncNext, set FeatureCaptureResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncSetStatusCode = "set_status_code"

	// FuncGetResponseBody writes the body written by FuncNext to memory if it
	// exists and isn't larger than `buf-limit`. The result is its length in
	// bytes. This requires FeatureCaptureResponse.
	//
	// To enable FeatureCaptureResponse, FuncEnableFeatures prior to FuncNext.
	// Doing otherwise, or calling before FuncNext will panic.
	//
	// TODO: document on http-wasm-abi
	FuncGetResponseBody = "get_response_body"

	// FuncSetResponseBody overwrites the response body with a value read from
	// memory. In doing so, this overwrites the "Content-Length" header with
	// its length.
	//
	// To use this function after FuncNext, set FeatureCaptureResponse via
	// FuncEnableFeatures. Otherwise, this can be called when FuncNext wasn't.
	//
	// TODO: document on http-wasm-abi
	FuncSetResponseBody = "set_response_body"
)
