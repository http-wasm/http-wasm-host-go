package wasm

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var (
	testCtx = context.Background()

	// compile-time check to ensure host implements handler.Host.
	_ handler.Host = host{}
	// compile-time check to ensure guest implements http.Handler.
	_ http.Handler = &guest{}
)

func Test_host_GetURI(t *testing.T) {
	tests := []struct {
		name        string
		url         *url.URL
		expectedURI string
	}{
		{
			name: "coerces empty path to slash",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "",
			},
			expectedURI: "/",
		},
		{
			name: "encodes space",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/a b",
			},
			expectedURI: "/a%20b",
		},
		{
			name: "encodes query",
			url: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Path:     "/a b",
				RawQuery: "q=go+language",
			},
			expectedURI: "/a%20b?q=go+language",
		},
		{
			name: "double slash path",
			url: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "//foo",
			},
			expectedURI: "//foo",
		},
		{
			name: "empty query",
			url: &url.URL{
				Scheme:     "http",
				Host:       "example.com",
				Path:       "/foo",
				ForceQuery: true,
			},
			expectedURI: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: tc.url}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})
			if want, have := tc.expectedURI, h.GetURI(ctx); want != have {
				t.Errorf("unexpected uri, want: %s, have: %s", want, have)
			}
		})
	}
}

func Test_host_SetURI(t *testing.T) {
	tests := []struct {
		name        string
		expectedURI string
	}{
		{
			name:        "coerces empty path to slash",
			expectedURI: "/",
		},
		{
			name:        "encodes space",
			expectedURI: "/a%20b",
		},
		{
			name:        "encodes query",
			expectedURI: "/a%20b?q=go+language",
		},
		{
			name:        "double slash path",
			expectedURI: "//foo",
		},
		{
			name:        "empty query",
			expectedURI: "/foo?",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: &url.URL{}}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})
			h.SetURI(ctx, tc.expectedURI)
			if want, have := tc.expectedURI, r.URL.RequestURI(); want != have {
				t.Errorf("unexpected uri, want: %s, have: %s", want, have)
			}
		})
	}
}
