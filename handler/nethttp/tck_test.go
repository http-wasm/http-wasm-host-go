package wasm

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/tck"
)

func TestTCK(t *testing.T) {
	// Initialize the TCK guest wasm module.
	mw, err := NewMiddleware(context.Background(), tck.GuestWASM)
	if err != nil {
		t.Fatal(err)
	}

	// Set the delegate handler of the middleware to the backend.
	h := mw.NewHandler(context.Background(), tck.BackendHandler())

	tests := []struct {
		http2    bool
		expected string
	}{
		{
			http2:    false,
			expected: "HTTP/1.1",
		},
		{
			http2:    true,
			expected: "HTTP/2.0",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.expected, func(t *testing.T) {
			// Start the server.
			ts := httptest.NewUnstartedServer(h)
			if tc.http2 {
				ts.EnableHTTP2 = true
				ts.StartTLS()
			} else {
				ts.Start()
			}
			defer ts.Close()

			// Run tests, issuing HTTP requests to server.
			tck.Run(t, ts.Client(), ts.URL)
		})
	}
}
