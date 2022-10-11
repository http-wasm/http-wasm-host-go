package wasm

import (
	"io"
	"net/http"
)

// compile-time check to ensure bufferingRequestBody implements io.ReadCloser.
var _ io.ReadCloser = &bufferingRequestBody{}

// compile-time check to ensure bufferingResponseWriter implements
// http.ResponseWriter.
var _ http.ResponseWriter = &bufferingResponseWriter{}
