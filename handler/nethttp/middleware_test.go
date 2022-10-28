package wasm_test

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api/handler"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

var (
	testCtx     = context.Background()
	noopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
)

func TestConfig(t *testing.T) {
	tests := []handler.Features{
		0,
		handler.FeatureBufferRequest,
		handler.FeatureBufferResponse,
		handler.FeatureTrailers,
		handler.FeatureBufferRequest | handler.FeatureBufferResponse,
		handler.FeatureBufferRequest | handler.FeatureBufferResponse | handler.FeatureTrailers,
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.String(), func(t *testing.T) {
			guestConfig := make([]byte, 8)
			binary.LittleEndian.PutUint64(guestConfig, uint64(tc))
			mw, err := wasm.NewMiddleware(testCtx, test.BinExampleConfig, httpwasm.GuestConfig(guestConfig))
			if err != nil {
				t.Fatal(err)
			}
			defer mw.Close(testCtx)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// ensure the request body is readable
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if want, have := requestBody, string(body); want != have {
					t.Fatalf("unexpected request body, want: %q, have: %q", want, have)
				}
				r.Header.Set("Content-Type", "application/json")
				w.Write([]byte(responseBody)) // nolint
			})

			ts := httptest.NewServer(mw.NewHandler(testCtx, next))
			defer ts.Close()

			resp, err := ts.Client().Post(ts.URL, "application/json", strings.NewReader(requestBody))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// Ensure the response body was still readable!
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if want, have := responseBody, string(body); want != have {
				t.Fatalf("unexpected response body, want: %q, have: %q", want, have)
			}
		})
	}
}

func TestMethod(t *testing.T) {
	mw, err := wasm.NewMiddleware(testCtx, test.BinE2EMethod)
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, have := "POST", r.Method; want != have {
			t.Fatalf("unexpected request method, want: %q, have: %q", want, have)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if want, have := "GET", string(body); want != have {
			t.Fatalf("unexpected request body, want: %q, have: %q", want, have)
		}
	})

	ts := httptest.NewServer(mw.NewHandler(testCtx, next))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func TestURI(t *testing.T) {
	mw, err := wasm.NewMiddleware(testCtx, test.BinE2EURI)
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	var ts *httptest.Server
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, have := "/v1.0/hello?name=teddy", r.URL.RequestURI(); want != have {
			t.Fatalf("unexpected request URI, want: %q, have: %q", want, have)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if want, have := "/v1.0/hi?name=panda", string(body); want != have {
			t.Fatalf("unexpected request body, want: %q, have: %q", want, have)
		}

		w.Write([]byte(responseBody))
	})

	ts = httptest.NewServer(mw.NewHandler(testCtx, next))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/v1.0/hi?name=panda")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := responseBody, string(body); want != have {
		t.Fatalf("unexpected response body, want: %q, have: %q", want, have)
	}
}

func TestProtocolVersion(t *testing.T) {
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
			mw, err := wasm.NewMiddleware(testCtx, test.BinE2EProtocolVersion)
			if err != nil {
				t.Fatal(err)
			}
			defer mw.Close(testCtx)

			ts := httptest.NewUnstartedServer(mw.NewHandler(testCtx, noopHandler))
			if tc.http2 {
				ts.EnableHTTP2 = true
				ts.StartTLS()
			} else {
				ts.Start()
			}
			defer ts.Close()

			resp, err := ts.Client().Get(ts.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if want, have := tc.expected, string(body); want != have {
				t.Fatalf("unexpected response body, want: %q, have: %q", want, have)
			}
		})
	}
}

// TestHeaderNames uses test.BinE2EHeaderNames which ensures count/len are
// correct.
func TestHeaderNames(t *testing.T) {
	mw, err := wasm.NewMiddleware(testCtx, test.BinE2EHeaderNames)
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	ts := httptest.NewServer(mw.NewHandler(testCtx, noopHandler))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("invalid status code: %d, status message: %s", resp.StatusCode, resp.Status)
	}
}

// TestHandleResponse uses test.BinE2EHandleResponse which ensures reqCtx
// propagates from handler.FuncHandleRequest to handler.FuncHandleResponse.
func TestHandleResponse(t *testing.T) {
	mw, err := wasm.NewMiddleware(testCtx, test.BinE2EHandleResponse)
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	ts := httptest.NewServer(mw.NewHandler(testCtx, noopHandler))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("invalid status code: %d, status message: %s", resp.StatusCode, resp.Status)
	}
}
