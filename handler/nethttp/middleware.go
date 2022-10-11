package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

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

type host struct{}

// requestStateKey is a context.Context value associated with a requestState
// pointer to the current request.
type requestStateKey struct{}

type requestState struct {
	w          http.ResponseWriter
	r          *http.Request
	next       http.Handler
	calledNext bool
	features   handler.Features
}

func (s *requestState) enableFeatures(features handler.Features) {
	s.features = s.features.WithEnabled(features)
	if !s.features.IsEnabled(handler.FeatureBufferResponse) {
		return
	}
	if _, ok := s.w.(*bufferingResponseWriter); !ok { // don't double-wrap
		s.w = &bufferingResponseWriter{delegate: s.w}
	}
}

func (s *requestState) handleNext() {
	if s.calledNext {
		panic("already called next")
	}
	s.calledNext = true

	// If we set the intercepted the request body for any reason, reset it
	// before calling downstream.
	if br, ok := s.r.Body.(*bufferingRequestBody); ok {
		if br.buffer.Len() == 0 {
			s.r.Body = http.NoBody
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

// EnableFeatures implements the same method as documented on handler.Host.
func (h host) EnableFeatures(ctx context.Context, features handler.Features) (result handler.Features) {
	if s, ok := ctx.Value(requestStateKey{}).(*requestState); ok {
		s.enableFeatures(features)
		return s.features
	} else if i, ok := ctx.Value(internalhandler.InitStateKey{}).(*internalhandler.InitState); ok {
		i.Features = i.Features.WithEnabled(features)
		return i.Features
	} else {
		panic("unexpected context state")
	}
}

// GetURI implements the same method as documented on handler.Host.
func (h host) GetURI(ctx context.Context) string {
	r := requestStateFromContext(ctx).r
	u := r.URL
	result := u.EscapedPath()
	if result == "" {
		result = "/"
	}
	if u.ForceQuery || u.RawQuery != "" {
		result += "?" + u.RawQuery
	}
	return result
}

// SetURI implements the same method as documented on handler.Host.
func (h host) SetURI(ctx context.Context, uri string) {
	r := requestStateFromContext(ctx).r
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(err)
	}
	r.URL.RawPath = u.RawPath
	r.URL.Path = u.Path
	r.URL.ForceQuery = u.ForceQuery
	r.URL.RawQuery = u.RawQuery
}

// GetRequestHeader implements the same method as documented on handler.Host.
func (h host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	r := requestStateFromContext(ctx).r
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

// GetRequestBody implements the same method as documented on handler.Host.
func (h host) GetRequestBody(ctx context.Context) []byte {
	s := requestStateFromContext(ctx)
	defer s.r.Body.Close()
	if b, err := io.ReadAll(s.r.Body); err != nil {
		panic(err)
	} else {
		return b
	}
}

// SetRequestBody implements the same method as documented on handler.Host.
func (h host) SetRequestBody(ctx context.Context, body []byte) {
	s := requestStateFromContext(ctx)
	// TODO: verify if ownership transfer is ok or not.
	s.r.Body = io.NopCloser(bytes.NewBuffer(body))
}

// GetStatusCode implements the same method as documented on handler.Host.
func (h host) GetStatusCode(ctx context.Context) uint32 {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		return w.statusCode
	}
	panic(fmt.Errorf("can't read back status code unless %s is enabled",
		handler.FeatureBufferResponse))
}

// SetStatusCode implements the same method as documented on handler.Host.
func (h host) SetStatusCode(ctx context.Context, statusCode uint32) {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.statusCode = statusCode
	} else if !s.calledNext {
		s.w.WriteHeader(int(statusCode))
	} else {
		panic("already called next")
	}
}

// SetResponseHeader implements the same method as documented on handler.Host.
func (h host) SetResponseHeader(ctx context.Context, name, value string) {
	s := requestStateFromContext(ctx)
	if s.calledNext && !s.features.IsEnabled(handler.FeatureBufferResponse) {
		panic("already called next")
	}
	s.w.Header().Set(name, value)
}

// GetResponseBody implements the same method as documented on handler.Host.
func (h host) GetResponseBody(ctx context.Context) []byte {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		return w.body
	}
	panic(fmt.Errorf("can't read back response body unless %s is enabled",
		handler.FeatureBufferResponse))
}

// SetResponseBody implements the same method as documented on handler.Host.
func (h host) SetResponseBody(ctx context.Context, body []byte) {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.body = body
	} else if !s.calledNext {
		s.w.Write(body) // nolint
	} else {
		panic("already called next")
	}
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
	features := (host{}).EnableFeatures(ctx, g.features)
	if features.IsEnabled(handler.FeatureBufferRequest) {
		s.r.Body = &bufferingRequestBody{delegate: r.Body}
	}
	if err := g.handle(ctx); err != nil {
		// TODO: after testing, shouldn't send errors into the HTTP response.
		w.WriteHeader(500)
		w.Write([]byte(err.Error())) // nolint
	} else if bw, ok := s.w.(*bufferingResponseWriter); ok {
		bw.release()
	}
}
