package wasm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"

	handlerapi "github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/handler"
)

// compile-time checks to ensure interfaces are implemented.
var _ http.Handler = (*guest)(nil)

type Middleware handlerapi.Middleware[http.Handler]

type middleware struct {
	m handler.Middleware
}

func NewMiddleware(ctx context.Context, guest []byte, options ...handler.Option) (Middleware, error) {
	m, err := handler.NewMiddleware(ctx, guest, host{}, options...)
	if err != nil {
		return nil, err
	}

	return &middleware{m: m}, nil
}

// requestStateKey is a context.Context value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

type requestState struct {
	w        http.ResponseWriter
	r        *http.Request
	next     http.Handler
	features handlerapi.Features
}

func newRequestState(w http.ResponseWriter, r *http.Request, g *guest) *requestState {
	s := &requestState{w: w, r: r, next: g.next}
	s.enableFeatures(g.features)
	return s
}

func (s *requestState) enableFeatures(features handlerapi.Features) {
	s.features = s.features.WithEnabled(features)
	if features.IsEnabled(handlerapi.FeatureBufferRequest) {
		s.r.Body = &bufferingRequestBody{delegate: s.r.Body}
	}
	if s.features.IsEnabled(handlerapi.FeatureBufferResponse) {
		if _, ok := s.w.(*bufferingResponseWriter); !ok { // don't double-wrap
			s.w = &bufferingResponseWriter{delegate: s.w}
		}
	}
}

func (s *requestState) handleNext() (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if e, ok := recovered.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", recovered)
			}
		}
	}()

	// If we set the intercepted the request body for any reason, reset it
	// before calling downstream.
	if br, ok := s.r.Body.(*bufferingRequestBody); ok {
		if br.buffer.Len() == 0 {
			s.r.Body = br.delegate
		} else {
			br.Close() // nolint
			s.r.Body = io.NopCloser(&br.buffer)
		}
	}
	s.next.ServeHTTP(s.w, s.r)
	return
}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

// NewHandler implements the same method as documented on handler.Middleware.
func (w *middleware) NewHandler(_ context.Context, next http.Handler) http.Handler {
	h := &guest{
		handleRequest:  w.m.HandleRequest,
		handleResponse: w.m.HandleResponse,
		next:           next,
		features:       w.m.Features(),
	}
	runtime.SetFinalizer(h, func(h *guest) {
		if err := w.Close(context.Background()); err != nil {
			fmt.Printf("[http-wasm-host-go] middleware Close failed: %v", err)
		}
	})
	return h
}

// Close implements the same method as documented on handler.Middleware.
func (w *middleware) Close(ctx context.Context) error {
	return w.m.Close(ctx)
}

type guest struct {
	handleRequest  func(ctx context.Context) (outCtx context.Context, ctxNext handlerapi.CtxNext, err error)
	handleResponse func(ctx context.Context, reqCtx uint32, err error) error
	next           http.Handler
	features       handlerapi.Features
}

// ServeHTTP implements http.Handler
func (g *guest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	s := newRequestState(w, r, g)
	ctx := context.WithValue(r.Context(), requestStateKey{}, s)
	outCtx, ctxNext, requestErr := g.handleRequest(ctx)
	if requestErr != nil {
		handleErr(w, requestErr)
	}

	// If buffering was enabled, ensure it flushes.
	if bw, ok := s.w.(*bufferingResponseWriter); ok {
		defer bw.release()
	}

	// Returning zero means the guest wants to break the handler chain, and
	// handle the response directly.
	if uint32(ctxNext) == 0 {
		return
	}

	// Otherwise, the host calls the next handler.
	err := s.handleNext()

	// Finally, call the guest with the response or error
	if err = g.handleResponse(outCtx, uint32(ctxNext>>32), err); err != nil {
		panic(err)
	}
}

func handleErr(w http.ResponseWriter, requestErr error) {
	// TODO: after testing, shouldn't send errors into the HTTP response.
	w.WriteHeader(500)
	w.Write([]byte(requestErr.Error())) // nolint
}
