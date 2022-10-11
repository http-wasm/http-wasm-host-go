package wasm

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

// compile-time check to ensure host implements handler.Host.
var _ handler.Host = host{}

// compile-time check to ensure guest implements http.Handler.
var _ http.Handler = &guest{}

func TestConfig(t *testing.T) {
	requestBody := "{\"hello\": \"panda\"}"
	responseBody := "{\"hello\": \"world\"}"

	tests := []handler.Features{
		0,
		handler.FeatureBufferRequest,
		handler.FeatureBufferResponse,
		handler.FeatureBufferRequest | handler.FeatureBufferResponse,
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.String(), func(t *testing.T) {
			ctx := context.Background()
			guestConfig := make([]byte, 8)
			binary.LittleEndian.PutUint64(guestConfig, uint64(tc))
			mw, err := NewMiddleware(ctx, test.ConfigWasm, httpwasm.GuestConfig(guestConfig))
			if err != nil {
				log.Panicln(err)
			}
			defer mw.Close(ctx)

			// Create the real request handler.
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// ensure the request body is readable
				body, err := io.ReadAll(r.Body)
				if err != nil {
					log.Panicln(err)
				}
				if want, have := requestBody, string(body); want != have {
					log.Panicf("unexpected request body, want: %q, have: %q", want, have)
				}
				r.Header.Set("Content-Type", "application/json")
				w.Write([]byte(responseBody)) // nolint
			})

			// Wrap this with an interceptor implemented in WebAssembly.
			wrapped := mw.NewHandler(ctx, next)

			// Start the server with the wrapped handler.
			ts := httptest.NewServer(wrapped)
			defer ts.Close()

			// Make a client request and print the contents to the same logger
			resp, err := ts.Client().Post(ts.URL, "application/json", strings.NewReader(requestBody))
			if err != nil {
				log.Panicln(err)
			}
			defer resp.Body.Close()

			// Ensure the response body was still readable!
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Panicln(err)
			}
			if want, have := responseBody, string(body); want != have {
				log.Panicf("unexpected response body, want: %q, have: %q", want, have)
			}
		})
	}
}

func Test_GetURI(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), requestStateKey{}, &requestState{r: r})
			if want, have := tc.expectedURI, h.GetURI(ctx); want != have {
				t.Errorf("unexpected uri, want: %s, have: %s", want, have)
			}
		})
	}
}

func Test_SetURI(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), requestStateKey{}, &requestState{r: r})
			h.SetURI(ctx, tc.expectedURI)
			if want, have := tc.expectedURI, r.URL.RequestURI(); want != have {
				t.Errorf("unexpected uri, want: %s, have: %s", want, have)
			}
		})
	}
}
