package wasm

import (
	"context"
	"net/http"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

// Handler is a http.Handler implemented by a Wasm module. `ServeHTTP` is
// dispatched to handler.FuncHandle, which is exported by the guest.
type Handler interface {
	http.Handler
	api.Closer
}

type Middleware handler.Middleware[http.Handler, Handler]

type middleware struct {
	runtime *internalhandler.Runtime
}

func NewMiddleware(ctx context.Context, guest []byte, options ...httpwasm.Option) (Middleware, error) {
	r, err := internalhandler.NewRuntime(ctx, guest, &host{}, options...)
	if err != nil {
		return nil, err
	}
	return &middleware{runtime: r}, nil
}

type host struct{}

// requestStateKey is a context.Context Value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

type requestState struct {
	request    *http.Request
	response   http.ResponseWriter
	handleNext func()
}

func withRequestState(ctx context.Context, response http.ResponseWriter, request *http.Request, next http.Handler) context.Context {
	return context.WithValue(ctx, requestStateKey{}, &requestState{
		request:    request,
		response:   response,
		handleNext: func() { next.ServeHTTP(response, request) },
	})
}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

// GetRequestHeader implements the same method as documented on
// handler.Host.
func (h host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	r := requestStateFromContext(ctx).request
	if values := r.Header.Values(name); len(values) == 0 {
		return "", false
	} else {
		return values[0], true
	}
}

// Next implements the same method as documented on handler.Host.
func (h host) Next(ctx context.Context) {
	requestStateFromContext(ctx).handleNext()
}

// SetResponseHeader implements the same method as documented on handler.Host.
func (h host) SetResponseHeader(ctx context.Context, name, value string) {
	r := requestStateFromContext(ctx).response
	r.Header().Set(name, value)
}

// SendResponse implements the same method as documented on handler.Host.
func (h host) SendResponse(ctx context.Context, statusCode uint32, body []byte) {
	r := requestStateFromContext(ctx).response
	r.WriteHeader(int(statusCode))
	if body != nil {
		r.Write(body) // nolint
	}
}

// NewHandler implements the same method as documented on handler.Middleware.
func (w *middleware) NewHandler(ctx context.Context, next http.Handler) (Handler, error) {
	g, err := w.runtime.NewGuest(ctx)
	if err != nil {
		return nil, err
	}

	return &guest{guest: g, next: next}, nil
}

// Close implements the same method as documented on handler.Middleware.
func (w *middleware) Close(ctx context.Context) error {
	return w.runtime.Close(ctx)
}

// compile-time check to ensure guest implements Handler.
var _ Handler = &guest{}

type guest struct {
	guest *internalhandler.Guest
	next  http.Handler
}

// ServeHTTP implements http.Handler
func (w *guest) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	ctx := withRequestState(request.Context(), response, request, w.next)
	if err := w.guest.Handle(ctx); err != nil {
		// TODO: after testing, shouldn't send errors into the HTTP response.
		response.Write([]byte(err.Error())) // nolint
		response.WriteHeader(500)
	}
}

// Close implements api.Closer
func (w *guest) Close(ctx context.Context) error {
	return w.guest.Close(ctx)
}
