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
)

var testCtx = context.Background()

func Test_host_GetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			r := &http.Request{Method: tc}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

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
			r := &http.Request{URL: &url.URL{}}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

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
			r := &http.Request{URL: tc.url}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

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
			r := &http.Request{URL: &url.URL{}}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

			h.SetURI(ctx, tc.input)
			if want, have := tc.expected, r.URL.RequestURI(); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetRequestHeaderNames(t *testing.T) {
	r := &http.Request{Header: testHeaders.Clone()}
	ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

	want := []string{"Content-Type", "Vary", "Empty"}
	have := host{}.GetRequestHeaderNames(ctx)
	sort.Strings(have)
	if reflect.DeepEqual(want, have) {
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
			name:       "multi-field",
			headerName: "Set-Cookie",
			expected:   []string{"a", "b"},
		},
		{
			name:       "comma value",
			headerName: "Vary",
			expected:   []string{"Accept-Encoding, User-Agent"},
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
			r := &http.Request{Header: testHeaders.Clone()}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

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
		value      string
	}{
		{
			name:       "single value",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "single value overwrites",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "multi-field overwrites",
			headerName: "Set-Cookie",
			value:      "z",
		},
		{
			name:       "comma value",
			headerName: "X-Forwarded-For",
			value:      "1.2.3.4, 4.5.6.7",
		},
		{
			name:       "comma value overwrites",
			headerName: "Vary",
			value:      "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "aloha",
		},
		{
			name:       "empty value overwrites",
			headerName: "Empty",
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
			r := &http.Request{Header: testHeaders.Clone()}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

			h.SetRequestHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.value, strings.Join(r.Header.Values(tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetResponseHeaderNames(t *testing.T) {
	w := &httptest.ResponseRecorder{HeaderMap: testHeaders}
	ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{w: w})

	want := []string{"Content-Type", "Vary", "Empty"}
	have := host{}.GetResponseHeaderNames(ctx)
	sort.Strings(have)
	if reflect.DeepEqual(want, have) {
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
			name:       "multi-field",
			headerName: "Set-Cookie",
			expected:   []string{"a", "b"},
		},
		{
			name:       "comma value",
			headerName: "Vary",
			expected:   []string{"Accept-Encoding, User-Agent"},
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
			w := &httptest.ResponseRecorder{HeaderMap: testHeaders}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{w: w})

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
		value      string
	}{
		{
			name:       "single value",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "single value overwrites",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "multi-field overwrites",
			headerName: "Set-Cookie",
			value:      "z",
		},
		{
			name:       "comma value",
			headerName: "X-Forwarded-For",
			value:      "1.2.3.4, 4.5.6.7",
		},
		{
			name:       "comma value overwrites",
			headerName: "Vary",
			value:      "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "aloha",
		},
		{
			name:       "empty value overwrites",
			headerName: "Empty",
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
			w := &httptest.ResponseRecorder{HeaderMap: testHeaders}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{w: w})

			h.SetResponseHeaderValue(ctx, tc.headerName, tc.value)
			if want, have := tc.value, strings.Join(w.Header().Values(tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

// Note: senders are supposed to concatenate multiple fields with the same
// name on comma, except the response header Set-Cookie. That said, a lot
// of middleware don't know about this and may repeat other headers anyway.
// See https://www.rfc-editor.org/rfc/rfc9110#section-5.2
var testHeaders = http.Header{
	"Content-Type": []string{"text/plain"},
	"Set-Cookie":   []string{"a", "b"},
	"Vary":         []string{"Accept-Encoding, User-Agent"},
	"Empty":        []string{""},
}
