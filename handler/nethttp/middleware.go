package wasm

import (
	"context"
	"net/http"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

type Middleware handler.Middleware[http.Handler]

type middleware struct {
	runtime *internalhandler.Runtime
	// TODO: pool
	guest *internalhandler.Guest
}

func NewMiddleware(ctx context.Context, guest []byte, options ...httpwasm.Option) (Middleware, error) {
	r, err := internalhandler.NewRuntime(ctx, guest, &host{}, options...)
	if err != nil {
		return nil, err
	}
	g, err := r.NewGuest(ctx)
	if err != nil {
		return nil, err
	}
	return &middleware{runtime: r, guest: g}, nil
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

// GetPath implements the same method as documented on handler.Host.
func (h host) GetPath(ctx context.Context) string {
	r := requestStateFromContext(ctx).request
	return r.URL.Path
}

// SetPath implements the same method as documented on handler.Host.
func (h host) SetPath(ctx context.Context, path string) {
	r := requestStateFromContext(ctx).request
	r.URL.Path = path
}

// GetRequestHeader implements the same method as documented on handler.Host.
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
func (w *middleware) NewHandler(ctx context.Context, next http.Handler) http.Handler {
	return &guest{handle: w.guest.Handle, next: next}
}

// Close implements the same method as documented on handler.Middleware.
func (w *middleware) Close(ctx context.Context) error {
	return w.runtime.Close(ctx)
}

type guest struct {
	handle func(ctx context.Context) (err error)
	next   http.Handler
}

// ServeHTTP implements http.Handler
func (w *guest) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	ctx := withRequestState(request.Context(), response, request, w.next)
	if err := w.handle(ctx); err != nil {
		// TODO: after testing, shouldn't send errors into the HTTP response.
		response.Write([]byte(err.Error())) // nolint
		response.WriteHeader(500)
	}
}
