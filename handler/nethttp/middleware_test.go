package wasm

import "github.com/http-wasm/http-wasm-host-go/api/handler"

// compile-time check to ensure host implements handler.Host.
var _ handler.Host = host{}

// compile-time check to ensure guest implements Handler.
var _ Handler = &guest{}
