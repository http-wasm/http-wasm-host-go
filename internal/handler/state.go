package internalhandler

import (
	"context"
	"io"
	"net/http"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

// requestStateKey is a context.Context value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

type requestState struct {
	calledNext         bool
	requestBodyReader  io.ReadCloser
	requestBodyWriter  io.Writer
	responseBodyReader io.ReadCloser
	responseBodyWriter io.Writer
	features           handler.Features
}

func (r *requestState) closeRequest() (err error) {
	if reqBW := r.requestBodyWriter; reqBW != nil {
		if f, ok := reqBW.(http.Flusher); ok {
			f.Flush()
		}
		r.requestBodyWriter = nil
	}
	if reqBR := r.requestBodyReader; reqBR != nil {
		err = reqBR.Close()
		r.requestBodyReader = nil
	}
	return
}

// Close implements io.Closer
func (r *requestState) Close() (err error) {
	err = r.closeRequest()
	if respBW := r.responseBodyWriter; respBW != nil {
		if f, ok := respBW.(http.Flusher); ok {
			f.Flush()
		}
		r.responseBodyWriter = nil
	}
	if respBR := r.responseBodyReader; respBR != nil {
		err = respBR.Close()
		r.responseBodyReader = nil
	}
	return
}
