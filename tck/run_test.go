package tck_test

import (
	"context"
	"net/http/httptest"
	"testing"

	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/http-wasm/http-wasm-host-go/tck"
)

func TestTCK(t *testing.T) {
	// Initialize the TCK guest wasm module.
	mw, err := wasm.NewMiddleware(context.Background(), tck.GuestWASM)
	if err != nil {
		t.Fatal(err)
	}
	// Set the delegate handler of the middleware to the backend.
	h := mw.NewHandler(context.Background(), tck.BackendHandler())
	// Start the server.
	server := httptest.NewServer(h)

	// Run tests, issuing HTTP requests to server.
	tck.Run(t, server.URL)
}
