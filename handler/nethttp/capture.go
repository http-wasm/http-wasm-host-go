package wasm

import (
	"net/http"
)

type capturingResponseWriter struct {
	delegate   http.ResponseWriter
	statusCode uint32
	body       []byte
}

// Header dispatches to the delegate.
func (w *capturingResponseWriter) Header() http.Header {
	return w.delegate.Header()
}

// Write captures the response body.
func (w *capturingResponseWriter) Write(bytes []byte) (int, error) {
	w.body = bytes
	return len(bytes), nil
}

// WriteHeader captures the status code.
func (w *capturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = uint32(statusCode)
}

// release sends any response data collected.
func (w *capturingResponseWriter) release() {
	// If we deferred the response, release it.
	if statusCode := w.statusCode; statusCode != 0 {
		w.delegate.WriteHeader(int(statusCode))
	}
	if body := w.body; len(body) != 0 {
		w.delegate.Write(body) // nolint
	}
}
