package test

import (
	_ "embed"
	"log"
	"os"
	"path"
	"runtime"
)

//go:embed testdata/bench/log.wasm
var BinBenchLog []byte

//go:embed testdata/bench/get_uri.wasm
var BinBenchGetURI []byte

//go:embed testdata/bench/set_uri.wasm
var BinBenchSetURI []byte

//go:embed testdata/bench/get_header_names.wasm
var BinBenchGetHeaderNames []byte

//go:embed testdata/bench/get_header_values.wasm
var BinBenchGetHeaderValues []byte

//go:embed testdata/bench/set_header_value.wasm
var BinBenchSetHeaderValue []byte

//go:embed testdata/bench/add_header_value.wasm
var BinBenchAddHeaderValue []byte

//go:embed testdata/bench/remove_header.wasm
var BinBenchRemoveHeader []byte

//go:embed testdata/bench/read_body.wasm
var BinBenchReadBody []byte

//go:embed testdata/bench/read_body_stream.wasm
var BinBenchReadBodyStream []byte

//go:embed testdata/bench/write_body.wasm
var BinBenchWriteBody []byte

//go:embed testdata/bench/set_status_code.wasm
var BinBenchSetStatusCode []byte

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

//go:embed testdata/e2e/handle_response.wasm
var BinE2EHandleResponse []byte

//go:embed testdata/e2e/header_names.wasm
var BinE2EHeaderNames []byte

// binExample instead of go:embed as files aren't relative to this directory.
func binExample(name string) []byte {
	_, thisFile, _, ok := runtime.Caller(1)
	if !ok {
		log.Panicln("cannot determine current path")
	}
	p := path.Join(path.Dir(thisFile), "..", "..", "examples", name+".wasm")
	if wasm, err := os.ReadFile(p); err != nil {
		log.Panicln(err)
		return nil
	} else {
		return wasm
	}
}
