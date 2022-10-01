package wasm

import (
	"context"
	"strconv"

	"github.com/valyala/fasthttp"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

// RequestHandler is a fasthttp.RequestHandler implemented by a Wasm module.
// Handle dispatched to handler.FuncHandle, which is exported by the guest.
type RequestHandler interface {
	// Handle implements fasthttp.RequestHandler
	Handle(*fasthttp.RequestCtx)

	api.Closer
}

type Middleware handler.Middleware[fasthttp.RequestHandler, RequestHandler]

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

// GetRequestHeader implements the same method as documented on
// handler.Host.
func (h host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	r := &ctx.(*fasthttp.RequestCtx).Request
	if value := r.Header.Peek(name); value == nil {
		return "", false
	} else {
		return string(value), true
	}
}

// Next implements the same method as documented on handler.Host.
func (h host) Next(ctx context.Context) {
	fastCtx := ctx.(*fasthttp.RequestCtx)
	fastCtx.UserValue("next").(fasthttp.RequestHandler)(fastCtx)
}

// SetResponseHeader implements the same method as documented on handler.Host.
func (h host) SetResponseHeader(ctx context.Context, name, value string) {
	r := &ctx.(*fasthttp.RequestCtx).Response
	r.Header.Set(name, value)
}

// SendResponse implements the same method as documented on handler.Host.
func (h host) SendResponse(ctx context.Context, statusCode uint32, body []byte) {
	r := &ctx.(*fasthttp.RequestCtx).Response
	if body != nil {
		r.Header.Set("Content-Length", strconv.Itoa(len(body)))
		r.AppendBody(body)
	}
	r.SetStatusCode(int(statusCode))
}

// NewHandler implements the same method as documented on handler.Middleware.
func (w *middleware) NewHandler(ctx context.Context, next fasthttp.RequestHandler) (RequestHandler, error) {
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

type guest struct {
	guest *internalhandler.Guest
	next  fasthttp.RequestHandler
}

// Handle implements RequestHandler.Handle
func (w *guest) Handle(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue("next", w.next)
	if err := w.guest.Handle(ctx); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

// Close implements api.Closer
func (w *guest) Close(ctx context.Context) error {
	return w.guest.Close(ctx)
}
