[![Build](https://github.com/http-wasm/http-wasm-host-go/workflows/build/badge.svg)](https://github.com/http-wasm/http-wasm-host-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/http-wasm/http-wasm-host-go)](https://goreportcard.com/report/github.com/http-wasm/http-wasm-host-go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

# http-wasm Host Library for Go

[http-wasm][1] defines HTTP functions implemented in [WebAssembly][2]. This
repository includes [http-handler ABI][3] middleware for various HTTP server
libraries written in Go.

* [nethttp](handler/nethttp): [net/http Handler][4]
* [fasthttp](handler/fasthttp): [net/http RequestHandler][5]

# WARNING: This is a proof of concept!

The current maturity phase is proof of concept. Once this has a working example
in [dapr][6], we will go back and revisit things intentionally deferred.

Meanwhile, minor details and test coverage will fall short of production
standards. This helps us deliver the proof-of-concept faster and prevents
wasted energy in the case that the concept isn't acceptable at all.

[1]: https://github.com/http-wasm
[2]: https://webassembly.org/
[3]: https://github.com/http-wasm/http-wasm-abi/blob/main/http-handler/http-handler.wit.md
[4]: https://pkg.go.dev/net/http#Handler
[5]: https://github.com/valyala/fasthttp
[6]: https://github.com/http-wasm/components-contrib/
