package test

import (
	_ "embed"
)

//go:embed testdata/examples/auth.wasm
var BinExampleAuth []byte

//go:embed testdata/examples/config.wasm
var BinExampleConfig []byte

//go:embed testdata/examples/wasi.wasm
var BinExampleWASI []byte

//go:embed testdata/examples/log.wasm
var BinExampleLog []byte

//go:embed testdata/examples/router.wasm
var BinExampleRouter []byte

//go:embed testdata/examples/redact.wasm
var BinExampleRedact []byte

//go:embed testdata/tests/method.wasm
var BinTestMethod []byte

//go:embed testdata/tests/protocol_version.wasm
var BinTestProtocolVersion []byte
