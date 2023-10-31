package wasm

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/testing/handlertest"
)

var testCtx = context.Background()

func Test_host(t *testing.T) {
	newCtx := func(features handler.Features) (context.Context, handler.Features) {
		// The below configuration supports all features.
		r, _ := http.NewRequest("GET", "", bytes.NewReader(nil))
		r.RemoteAddr = "1.2.3.4:12345"
		w := &bufferingResponseWriter{delegate: &httptest.ResponseRecorder{HeaderMap: map[string][]string{}}}
		return context.WithValue(testCtx, requestStateKey{}, &requestState{r: r, w: w}), features
	}

	if err := handlertest.HostTest(t, host{}, newCtx); err != nil {
		t.Fatal(err)
	}
}

// Test_host_GetProtocolVersion ensures HTTP/2.0 is readable
func Test_host_GetProtocolVersion(t *testing.T) {
	tests := []string{"HTTP/1.1", "HTTP/2.0"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			r := &http.Request{Proto: tc}
			ctx := context.WithValue(testCtx, requestStateKey{}, &requestState{r: r})

			if want, have := tc, h.GetProtocolVersion(ctx); want != have {
				t.Errorf("unexpected protocolVersion, want: %v, have: %v", want, have)
			}
		})
	}
}
