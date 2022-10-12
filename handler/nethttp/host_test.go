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
		expected string
	}{
		{
			name:     "coerces empty path to slash",
			expected: "/",
		},
		{
			name:     "encodes space",
			expected: "/a%20b",
		},
		{
			name:     "encodes query",
			expected: "/a%20b?q=go+language",
		},
		{
			name:     "double slash path",
			expected: "//foo",
		},
		{
			name:     "empty query",
			expected: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: &url.URL{}}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

			h.SetURI(ctx, tc.expected)
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

func Test_host_GetRequestHeader(t *testing.T) {
	tests := []struct {
		name          string
		headerName    string
		expectedOk    bool
		expectedValue string
	}{
		{
			name:          "single value",
			headerName:    "Content-Type",
			expectedOk:    true,
			expectedValue: "text/plain",
		},
		{
			name:          "multi-field first value",
			headerName:    "Set-Cookie",
			expectedOk:    true,
			expectedValue: "a",
		},
		{
			name:          "comma value",
			headerName:    "Vary",
			expectedOk:    true,
			expectedValue: "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expectedOk: true,
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

			value, ok := h.GetRequestHeader(ctx, tc.headerName)
			if want, have := tc.expectedValue, value; want != have {
				t.Errorf("unexpected header value, want: %v, have: %v", want, have)
			}
			if want, have := tc.expectedOk, ok; want != have {
				t.Errorf("unexpected header ok, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetRequestHeader(t *testing.T) {
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

			h.SetRequestHeader(ctx, tc.headerName, tc.value)
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

func Test_host_GetResponseHeader(t *testing.T) {
	tests := []struct {
		name          string
		headerName    string
		expectedOk    bool
		expectedValue string
	}{
		{
			name:          "single value",
			headerName:    "Content-Type",
			expectedOk:    true,
			expectedValue: "text/plain",
		},
		{
			name:          "multi-field first value",
			headerName:    "Set-Cookie",
			expectedOk:    true,
			expectedValue: "a",
		},
		{
			name:          "comma value",
			headerName:    "Vary",
			expectedOk:    true,
			expectedValue: "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expectedOk: true,
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

			value, ok := h.GetResponseHeader(ctx, tc.headerName)
			if want, have := tc.expectedValue, value; want != have {
				t.Errorf("unexpected header value, want: %v, have: %v", want, have)
			}
			if want, have := tc.expectedOk, ok; want != have {
				t.Errorf("unexpected header ok, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetResponseHeader(t *testing.T) {
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

			h.SetResponseHeader(ctx, tc.headerName, tc.value)
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
