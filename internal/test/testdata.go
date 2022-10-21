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

//go:embed testdata/bench/next.wasm
var BinBenchNext []byte

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

//go:embed testdata/e2e/header_names.wasm
var BinE2EHeaderNames []byte

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

// Note: senders are supposed to concatenate multiple fields with the same
// name on comma, except the response header Set-Cookie. That said, a lot
// of middleware don't know about this and may repeat other headers anyway.
// See https://www.rfc-editoreqHeaders.org/rfc/rfc9110#section-5.2

var (
	RequestHeaders = map[string][]string{
		"Content-Type":    {"text/plain"},
		"Custom":          {"1"},
		"X-Forwarded-For": {"client, proxy1", "proxy2"},
		"Empty":           {""},
	}
	ResponseHeaders = map[string][]string{
		"Content-Type": {"text/plain"},
		"Custom":       {"1"},
		"Set-Cookie":   {"a=b, c=d", "e=f"},
		"Empty":        {""},
	}
)
