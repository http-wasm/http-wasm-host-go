package wasm

import (
	"net/http"
)

// compile-time check to ensure capturingResponseWriter implements
// http.ResponseWriter.
var _ http.ResponseWriter = &capturingResponseWriter{}
