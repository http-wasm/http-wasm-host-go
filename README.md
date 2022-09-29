[![Build](https://github.com/http-wasm/http-wasm-host-go/workflows/build/badge.svg)](https://github.com/http-wasm/http-wasm-host-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/http-wasm/http-wasm-host-go)](https://goreportcard.com/report/github.com/http-wasm/http-wasm-host-go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

# http-wasm Host Library for Go

[http-wasm][1] is HTTP server middleware implemented in [WebAssembly][2].
This repository includes [host ABI][3] middleware for various HTTP server
libraries written in Go.

* [nethttp](middleware/nethttp): [net/http handler][4]

[1]: https://github.com/http-wasm
[2]: https://webassembly.org/
[3]: https://github.com/http-wasm/http-wasm-abi
[4]: https://pkg.go.dev/net/http#Handler
