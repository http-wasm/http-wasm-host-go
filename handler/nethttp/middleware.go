package wasm

import (
	"context"
	"io"
	"net/http"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

// compile-time checks to ensure interfaces are implemented.
var _ http.Handler = (*guest)(nil)

type Middleware handler.Middleware[http.Handler]

type middleware struct {
	runtime *internalhandler.Runtime
}

func NewMiddleware(ctx context.Context, guest []byte, options ...httpwasm.Option) (Middleware, error) {
	r, err := internalhandler.NewRuntime(ctx, guest, host{}, options...)
	if err != nil {
		return nil, err
	}
	return &middleware{runtime: r}, nil
}

// requestStateKey is a context.Context value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

type requestState struct {
	w        http.ResponseWriter
	r        *http.Request
	next     http.Handler
	features handler.Features
}

func (s *requestState) enableFeatures(features handler.Features) {
	s.features = s.features.WithEnabled(features)
	if s.features.IsEnabled(handler.FeatureBufferResponse) {
		if _, ok := s.w.(*bufferingResponseWriter); !ok { // don't double-wrap
			s.w = &bufferingResponseWriter{delegate: s.w}
		}
	}
	if features.IsEnabled(handler.FeatureBufferRequest) {
		s.r.Body = &bufferingRequestBody{delegate: s.r.Body}
	}
}

func (s *requestState) handleNext() {
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
}

func requestStateFromContext(ctx context.Context) *requestState {
	return ctx.Value(requestStateKey{}).(*requestState)
}

// NewHandler implements the same method as documented on handler.Middleware.
func (w *middleware) NewHandler(_ context.Context, next http.Handler) http.Handler {
	return &guest{handle: w.runtime.Handle, next: next, features: w.runtime.Features}
}

// Close implements the same method as documented on handler.Middleware.
func (w *middleware) Close(ctx context.Context) error {
	return w.runtime.Close(ctx)
}

type guest struct {
	handle   func(ctx context.Context) (err error)
	next     http.Handler
	features handler.Features
}

// ServeHTTP implements http.Handler
func (g *guest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The guest Wasm actually handles the request. As it may call host
	// functions, we add context parameters of the current request.
	s := &requestState{w: w, r: r, next: g.next}
	ctx := context.WithValue(r.Context(), requestStateKey{}, s)
	(host{}).EnableFeatures(ctx, g.features)
	if err := g.handle(ctx); err != nil {
		// TODO: after testing, shouldn't send errors into the HTTP response.
		w.WriteHeader(500)
		w.Write([]byte(err.Error())) // nolint
	} else if bw, ok := s.w.(*bufferingResponseWriter); ok {
		bw.release()
	}
}
