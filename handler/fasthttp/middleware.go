package wasm

import (
	"context"
	"strconv"

	"github.com/valyala/fasthttp"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

type Middleware handler.Middleware[fasthttp.RequestHandler]

// compile-time check to ensure middleware implements Middleware.
var _ Middleware = &middleware{}

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
	requestCtx *fasthttp.RequestCtx
	next       fasthttp.RequestHandler
}

func withRequestState(ctx context.Context, requestCtx *fasthttp.RequestCtx, next fasthttp.RequestHandler) context.Context {
	return context.WithValue(ctx, requestStateKey{}, &requestState{
		requestCtx: requestCtx,
		next:       next,
	})
}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

// GetPath implements the same method as documented on handler.Host.
func (h host) GetPath(ctx context.Context) string {
	r := &requestStateFromContext(ctx).requestCtx.Request
	return string(r.URI().Path())
}

// SetPath implements the same method as documented on handler.Host.
func (h host) SetPath(ctx context.Context, path string) {
	r := &requestStateFromContext(ctx).requestCtx.Request
	r.URI().SetPath(path)
}

// GetRequestHeader implements the same method as documented on
// handler.Host.
func (h host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	r := &requestStateFromContext(ctx).requestCtx.Request
	if value := r.Header.Peek(name); value == nil {
		return "", false
	} else {
		return string(value), true
	}
}

// Next implements the same method as documented on handler.Host.
func (h host) Next(ctx context.Context) {
	s := requestStateFromContext(ctx)
	s.next(s.requestCtx)
}

// SetResponseHeader implements the same method as documented on handler.Host.
func (h host) SetResponseHeader(ctx context.Context, name, value string) {
	r := &requestStateFromContext(ctx).requestCtx.Response
	r.Header.Set(name, value)
}

// SendResponse implements the same method as documented on handler.Host.
func (h host) SendResponse(ctx context.Context, statusCode uint32, body []byte) {
	r := &requestStateFromContext(ctx).requestCtx.Response
	if body != nil {
		r.Header.Set("Content-Length", strconv.Itoa(len(body)))
		r.AppendBody(body)
	}
	r.SetStatusCode(int(statusCode))
}

// NewHandler implements the same method as documented on handler.Middleware.
func (w *middleware) NewHandler(ctx context.Context, next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return (&guest{handle: w.runtime.Handle, next: next}).Handle
}

// Close implements the same method as documented on handler.Middleware.
func (w *middleware) Close(ctx context.Context) error {
	return w.runtime.Close(ctx)
}

type guest struct {
	handle func(ctx context.Context) (err error)
	next   fasthttp.RequestHandler
}

// Handle implements RequestHandler.Handle
func (w *guest) Handle(requestCtx *fasthttp.RequestCtx) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	ctx := withRequestState(requestCtx, requestCtx, w.next)
	if err := w.handle(ctx); err != nil {
		requestCtx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}
