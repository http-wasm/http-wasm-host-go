package internalhandler

import (
	"context"
	"io"

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
	responseBodyReader io.ReadCloser
	features           handler.Features
}

func (r *requestState) closeRequestBody() (err error) {
	if reqBR := r.requestBodyReader; reqBR != nil {
		err = reqBR.Close()
	}
	r.requestBodyReader = nil
	return
}

// Close implements io.Closer
func (r *requestState) Close() (err error) {
	err = r.closeRequestBody()
	if respBR := r.responseBodyReader; respBR != nil {
		err = respBR.Close()
	}
	r.responseBodyReader = nil
	return
}
