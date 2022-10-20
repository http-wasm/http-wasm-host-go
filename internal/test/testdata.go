package test

import (
	_ "embed"
	"log"
	"os"
	"path"
)

//go:embed testdata/bench/log.wasm
var BinBenchLog []byte

//go:embed testdata/bench/get_uri.wasm
var BinBenchGetURI []byte

//go:embed testdata/bench/set_uri.wasm
var BinBenchSetURI []byte

//go:embed testdata/bench/get_request_header_names.wasm
var BinBenchGetRequestHeaderNames []byte

//go:embed testdata/bench/get_request_header.wasm
var BinBenchGetRequestHeader []byte

//go:embed testdata/bench/read_request_body.wasm
var BinBenchReadRequestBody []byte

//go:embed testdata/bench/read_request_body_stream.wasm
var BinBenchReadRequestBodyStream []byte

//go:embed testdata/bench/next.wasm
var BinBenchNext []byte

//go:embed testdata/bench/set_status_code.wasm
var BinBenchSetStatusCode []byte

//go:embed testdata/bench/set_response_header.wasm
var BinBenchSetResponseHeader []byte

//go:embed testdata/bench/write_response_body.wasm
var BinBenchWriteResponseBody []byte

var BinExampleAuth = func() []byte {
	return binExample("auth")
}()

var BinExampleConfig = func() []byte {
	return binExample("config")
}()

var BinExampleLog = func() []byte {
	return binExample("log")
}()

var BinExampleRedact = func() []byte {
	return binExample("redact")
}()

var BinExampleRouter = func() []byte {
	return binExample("router")
}()

var BinExampleWASI = func() []byte {
	return binExample("wasi")
}()

//go:embed testdata/e2e/method.wasm
var BinE2EMethod []byte

//go:embed testdata/e2e/uri.wasm
var BinE2EURI []byte

//go:embed testdata/e2e/protocol_version.wasm
var BinE2EProtocolVersion []byte

// binExample instead of go:embed as files aren't relative to this directory.
func binExample(name string) []byte {
	p := path.Join("..", "..", "examples", name+".wasm")
	if wasm, err := os.ReadFile(p); err != nil {
		log.Panicln(err)
		return nil
	} else {
		return wasm
	}
}
