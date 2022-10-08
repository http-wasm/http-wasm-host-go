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

	// FuncGetPath writes the path to memory if it exists and isn't larger than
	// `buf-limit`. The result is length of the path in bytes.
	//
	// Note: The path does not include query parameters.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#get-path
	FuncGetPath = "get_path"

	// FuncSetPath Overwrites the request path with one read from memory.
	//
	// Note: The path does not include query parameters.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-path
	FuncSetPath = "set_path"

	// FuncNext calls a downstream handler and blocks until it is finished
	// processing.
	//
	// This is an alternative to calling FuncSetStatusCode or
	// FuncSetResponseBody to construct the response in guest wasm.
	//
	// Note: Whether the next handler sends the response is implementation
	// specific. Some implementations may flush the response before returning,
	// while others schedule it for later.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#next
	FuncNext = "next"

	// FuncSetResponseHeader overwrites a response header with a given name to
	// a value read from memory.
	//
	// See https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md#set-response-header
	FuncSetResponseHeader = "set_response_header"

	// FuncSetStatusCode overrides the status code. The default is 200.
	//
	// This is an alternative to calling FuncNext. Calling this afterwards has
	// undefined behavior.
	//
	// TODO: document on http-wasm-abi
	FuncSetStatusCode = "set_status_code"

	// FuncSetResponseBody overwrites the response body with a value read from
	// memory. In doing so, this overwrites the "Content-Length" header with
	// its length.
	//
	// This is an alternative to calling FuncNext. Calling this afterwards has
	// undefined behavior.
	//
	// TODO: document on http-wasm-abi
	FuncSetResponseBody = "set_response_body"
)
