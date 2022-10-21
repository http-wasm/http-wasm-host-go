package wasm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

var testCtx = context.Background()

func Test_host_GetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, r := newTestRequestContext()
			r.Method = tc

			if want, have := tc, h.GetMethod(ctx); want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, r := newTestRequestContext()

			h.SetMethod(ctx, tc)
			if want, have := tc, r.Method; want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetURI(t *testing.T) {
	tests := []struct {
		name     string
		url      *url.URL
		expected string
	}{
		{
			name: "coerces empty path to slash",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "",
			},
			expected: "/",
		},
		{
			name: "encodes space",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/a b",
			},
			expected: "/a%20b",
		},
		{
			name: "encodes query",
			url: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Path:     "/a b",
				RawQuery: "q=go+language",
			},
			expected: "/a%20b?q=go+language",
		},
		{
			name: "double slash path",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "//foo",
			},
			expected: "//foo",
		},
		{
			name: "empty query",
			url: &url.URL{
				Scheme:     "http",
				Host:       "example.com",
				Path:       "/foo",
				ForceQuery: true,
			},
			expected: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, r := newTestRequestContext()
			r.URL = tc.url

			if want, have := tc.expected, h.GetURI(ctx); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetURI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "coerces empty path to slash",
			expected: "/",
		},
		{
			name:     "encodes space",
			input:    "/a%20b",
			expected: "/a%20b",
		},
		{
			name:     "encodes query",
			input:    "/a%20b?q=go+language",
			expected: "/a%20b?q=go+language",
		},
		{
			name:     "double slash path",
			input:    "//foo",
			expected: "//foo",
		},
		{
			name:     "empty query",
			input:    "/foo?",
			expected: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, r := newTestRequestContext()

			h.SetURI(ctx, tc.input)
			if want, have := tc.expected, r.URL.RequestURI(); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetProtocolVersion(t *testing.T) {
	tests := []string{"HTTP/1.1", "HTTP/2.0"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx, r := newTestRequestContext()
			r.Proto = tc

			if want, have := tc, h.GetProtocolVersion(ctx); want != have {
				t.Errorf("unexpected protocolVersion, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetRequestHeaderNames(t *testing.T) {
	ctx, _ := newTestRequestContext()

	var want []string
	for k := range test.RequestHeaders {
		want = append(want, k)
	}
	sort.Strings(want)

	have := host{}.GetRequestHeaderNames(ctx)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetRequestHeaderValues(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			expected:   []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "X-Forwarded-For",
			expected:   []string{"client, proxy1", "proxy2"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expected:   []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			values := h.GetRequestHeaderValues(ctx, tc.headerName)
			if want, have := tc.expected, values; !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header values, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			expected:   "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			expected:   "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			expected:   "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "X-Forwarded-For",
			expected:   "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "X-Forwarded-For",
			expected:   "",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.SetRequestHeaderValue(ctx, tc.headerName, tc.expected)
			if want, have := tc.expected, strings.Join(h.GetRequestHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_AddRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		value      string
		expected   []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			expected:   []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			expected:   []string{""},
		},
		{
			name:       "existing",
			headerName: "X-Forwarded-For",
			value:      "proxy3",
			expected:   []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "lowercase",
			headerName: "x-forwarded-for",
			value:      "proxy3",
			expected:   []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "existing empty",
			headerName: "X-Forwarded-For",
			expected:   []string{"client, proxy1", "proxy2", ""},
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.AddRequestHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.expected, h.GetRequestHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_RemoveRequestHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
	}{
		{
			name:       "doesn't exist",
			headerName: "custom",
		},
		{
			name:       "empty",
			headerName: "Empty",
		},
		{
			name:       "exists",
			headerName: "Custom",
		},
		{
			name:       "lowercase",
			headerName: "custom",
		},
		{
			name:       "multi-field",
			headerName: "X-Forwarded-For",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestRequestContext()

			h.RemoveRequestHeader(ctx, tc.headerName)
			if have := h.GetRequestHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("unexpected headers: %v", have)
			}
		})
	}
}

func Test_host_GetResponseHeaderNames(t *testing.T) {
	ctx, _ := newTestResponseContext()

	var want []string
	for k := range test.ResponseHeaders {
		want = append(want, k)
	}
	sort.Strings(want)

	have := host{}.GetResponseHeaderNames(ctx)
	sort.Strings(have)
	if !reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetResponseHeaderValues(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			expected:   []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "Set-Cookie",
			expected:   []string{"a=b, c=d", "e=f"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expected:   []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			values := h.GetResponseHeaderValues(ctx, tc.headerName)
			if want, have := tc.expected, values; !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header values, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		expected   string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			expected:   "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			expected:   "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			expected:   "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "Set-Cookie",
			expected:   "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "Set-Cookie",
			expected:   "",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.SetResponseHeaderValue(ctx, tc.headerName, tc.expected)
			if want, have := tc.expected, strings.Join(h.GetResponseHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_AddResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		value      string
		expected   []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			expected:   []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			expected:   []string{""},
		},
		{
			name:       "existing",
			headerName: "Set-Cookie",
			value:      "g=h",
			expected:   []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "lowercase",
			headerName: "set-Cookie",
			value:      "g=h",
			expected:   []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "existing empty",
			headerName: "Set-Cookie",
			value:      "",
			expected:   []string{"a=b, c=d", "e=f", ""},
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.AddResponseHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.expected, h.GetResponseHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_RemoveResponseHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
	}{
		{
			name:       "doesn't exist",
			headerName: "new",
		},
		{
			name:       "empty",
			headerName: "Empty",
		},
		{
			name:       "exists",
			headerName: "Custom",
		},
		{
			name:       "lowercase",
			headerName: "custom",
		},
		{
			name:       "multi-field",
			headerName: "Set-Cookie",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestResponseContext()

			h.RemoveResponseHeader(ctx, tc.headerName)
			if have := h.GetResponseHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("unexpected headers: %v", have)
			}
		})
	}
}

func newTestRequestContext() (ctx context.Context, r *http.Request) {
	r = &http.Request{Proto: "HTTP/1.1", URL: &url.URL{}, Header: testRequestHeaders()}
	ctx = context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})
	return
}

func newTestResponseContext() (ctx context.Context, w http.ResponseWriter) {
	w = &httptest.ResponseRecorder{HeaderMap: testResponseHeaders()}
	ctx = context.WithValue(testCtx, requestStateKey{}, &requestState{w: w})
	return
}

func testRequestHeaders() http.Header {
	return testHeaders(test.RequestHeaders)
}

func testResponseHeaders() http.Header {
	return testHeaders(test.ResponseHeaders)
}

func testHeaders(t map[string][]string) (h http.Header) {
	h = make(http.Header, len(t))
	for k, vs := range t {
		// del first in case there is an existing default.
		h.Del(k)
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	return h
}
