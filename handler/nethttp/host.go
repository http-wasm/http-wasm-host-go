package wasm

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strings"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

type host struct{}

var _ handler.Host = host{}

// EnableFeatures implements the same method as documented on handler.Host.
func (h host) EnableFeatures(ctx context.Context, features handler.Features) {
	requestStateFromContext(ctx).enableFeatures(features)
}

// GetMethod implements the same method as documented on handler.Host.
func (h host) GetMethod(ctx context.Context) string {
	r := requestStateFromContext(ctx).r
	return r.Method
}

// SetMethod implements the same method as documented on handler.Host.
func (h host) SetMethod(ctx context.Context, method string) {
	r := requestStateFromContext(ctx).r
	r.Method = method
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

// GetProtocolVersion implements the same method as documented on handler.Host.
func (h host) GetProtocolVersion(ctx context.Context) string {
	r := requestStateFromContext(ctx).r
	return r.Proto
}

// GetRequestHeaderNames implements the same method as documented on handler.Host.
func (h host) GetRequestHeaderNames(ctx context.Context) (names []string) {
	r := requestStateFromContext(ctx).r
	count := len(r.Header)
	i := 0
	if r.Host != "" { // special-case the host header.
		count++
		names = make([]string, count)
		names[i] = "Host"
		i++
	} else {
		names = make([]string, count)
	}
	for n := range r.Header {
		names[i] = n
		i++
	}
	// Keys in a Go map don't have consistent ordering.
	sort.Strings(names)
	return
}

// GetRequestHeader implements the same method as documented on handler.Host.
func (h host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	r := requestStateFromContext(ctx).r
	if textproto.CanonicalMIMEHeaderKey(name) == "Host" { // special-case the host header.
		v := r.Host
		return v, v != ""
	}
	if values := r.Header.Values(name); len(values) == 0 {
		return "", false
	} else {
		return values[0], true
	}
}

// SetRequestHeader implements the same method as documented on handler.Host.
func (h host) SetRequestHeader(ctx context.Context, name, value string) {
	s := requestStateFromContext(ctx)
	s.r.Header.Set(name, value)
}

// ReadRequestBody implements the same method as documented on handler.Host.
func (h host) ReadRequestBody(ctx context.Context) io.ReadCloser {
	s := requestStateFromContext(ctx)
	return s.r.Body
}

// SetRequestBody implements the same method as documented on handler.Host.
func (h host) SetRequestBody(ctx context.Context, body []byte) {
	s := requestStateFromContext(ctx)
	// TODO: verify if ownership transfer is ok or not.
	s.r.Body = io.NopCloser(bytes.NewBuffer(body))
}

// Next implements the same method as documented on handler.Host.
func (h host) Next(ctx context.Context) {
	requestStateFromContext(ctx).handleNext()
}

// GetStatusCode implements the same method as documented on handler.Host.
func (h host) GetStatusCode(ctx context.Context) uint32 {
	s := requestStateFromContext(ctx)
	if statusCode := s.w.(*bufferingResponseWriter).statusCode; statusCode == 0 {
		return 200 // default
	} else {
		return statusCode
	}
}

// SetStatusCode implements the same method as documented on handler.Host.
func (h host) SetStatusCode(ctx context.Context, statusCode uint32) {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.statusCode = statusCode
	} else {
		s.w.WriteHeader(int(statusCode))
	}
}

// GetResponseHeaderNames implements the same method as documented on handler.Host.
func (h host) GetResponseHeaderNames(ctx context.Context) (names []string) {
	w := requestStateFromContext(ctx).w

	// allocate capacity == count though it might be smaller due to trailers.
	count := len(w.Header())
	names = make([]string, 0, count)

	for n := range w.Header() {
		if strings.HasPrefix(n, http.TrailerPrefix) {
			continue
		}
		names = append(names, n)
	}
	return
}

// GetResponseHeader implements the same method as documented on handler.Host.
func (h host) GetResponseHeader(ctx context.Context, name string) (string, bool) {
	w := requestStateFromContext(ctx).w
	if values := w.Header().Values(name); len(values) == 0 {
		return "", false
	} else {
		return values[0], true
	}
}

// SetResponseHeader implements the same method as documented on handler.Host.
func (h host) SetResponseHeader(ctx context.Context, name, value string) {
	s := requestStateFromContext(ctx)
	s.w.Header().Set(name, value)
}

// ReadResponseBody implements the same method as documented on handler.Host.
func (h host) ReadResponseBody(ctx context.Context) io.ReadCloser {
	s := requestStateFromContext(ctx)
	body := s.w.(*bufferingResponseWriter).body
	return io.NopCloser(bytes.NewReader(body))
}

// SetResponseBody implements the same method as documented on handler.Host.
func (h host) SetResponseBody(ctx context.Context, body []byte) {
	s := requestStateFromContext(ctx)
	if w, ok := s.w.(*bufferingResponseWriter); ok {
		w.body = body
	} else {
		s.w.Write(body) // nolint
	}
}
