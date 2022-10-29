[![Build](https://github.com/http-wasm/http-wasm-host-go/workflows/build/badge.svg)](https://github.com/http-wasm/http-wasm-host-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/http-wasm/http-wasm-host-go)](https://goreportcard.com/report/github.com/http-wasm/http-wasm-host-go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

# http-wasm Host Library for Go

[http-wasm][1] defines HTTP functions implemented in [WebAssembly][2]. This
repository includes [http_handler ABI][3] middleware for various HTTP server
libraries written in Go.

* [nethttp](handler/nethttp): [net/http Handler][4]

# WARNING: This is an early draft

The current maturity phase is early draft. Once this is integrated with
[coraza][5] and [dapr][6], we can begin discussions about compatability.

[1]: https://github.com/http-wasm
[2]: https://webassembly.org/
[3]: https://github.com/http-wasm/http-wasm-abi/blob/main/http_handler/http_handler.wit.md
[4]: https://pkg.go.dev/net/http#Handler
[5]: https://github.com/corazawaf/coraza-proxy-wasm
[6]: https://github.com/http-wasm/components-contrib/
