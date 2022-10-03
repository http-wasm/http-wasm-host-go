module github.com/http-wasm/http-wasm-host-go/handler/fasthttp

go 1.18

require (
	github.com/http-wasm/http-wasm-host-go v0.0.0
	github.com/valyala/fasthttp v1.40.0
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/tetratelabs/wazero v1.0.0-pre.2.0.20221003082636-0b4dbfd8d6ca // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
)

replace github.com/http-wasm/http-wasm-host-go => ../../
