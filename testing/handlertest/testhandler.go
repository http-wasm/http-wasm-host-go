// Package handlertest implements support for testing implementations
// of HTTP handlers.
//
// This is inspired by fstest.TestFS, but implemented differently, notably
// using a testing.T parameter for better reporting.
package handlertest

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

// HostTest tests a handler.Host by testing default property values and
// ability to change them.
//
// To use this, pass your host and the context which allows access to the
// request and response objects. This should be configured for "HTTP/1.1".
//
// Here's an example of a net/http host which supports no features:
//
//	newCtx := func(features handler.Features) (context.Context, handler.Features) {
//		if features != 0 {
//			return testCtx, 0 // unsupported
//		}
//
//		r, _ := http.NewRequest("GET", "", nil)
//		w := &httptest.ResponseRecorder{HeaderMap: map[string][]string{}}
//		return context.WithValue(testCtx, requestStateKey{}, &requestState{r: r, w: w}), features
//	}
//
//	// Run all the tests
//	handlertest.HostTest(t, host{}, newCtx)
func HostTest(t *testing.T, h handler.Host, newCtx func(handler.Features) (context.Context, handler.Features)) error {
	ht := hostTester{t: t, h: h, newCtx: newCtx}

	ht.testMethod()
	ht.testURI()
	ht.testProtocolVersion()
	ht.testRequestHeaders()
	ht.testRequestBody()
	ht.testRequestTrailers()
	ht.testStatusCode()
	ht.testResponseHeaders()
	ht.testResponseBody()
	ht.testResponseTrailers()

	if len(ht.errText) == 0 {
		return nil
	}
	return errors.New("TestHost found errors:\n" + string(ht.errText))
}

// A hostTester holds state for running the test.
type hostTester struct {
	t       *testing.T
	h       handler.Host
	newCtx  func(handler.Features) (context.Context, handler.Features)
	errText []byte
}

func (h *hostTester) testMethod() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetMethod default", func(t *testing.T) {
		// Check default
		if want, have := "GET", h.h.GetMethod(ctx); want != have {
			t.Errorf("unexpected default method, want: %v, have: %v", want, have)
		}
	})

	h.t.Run("SetMethod", func(t *testing.T) {
		for _, want := range []string{"POST", "OPTIONS"} {
			h.h.SetMethod(ctx, want)

			if have := h.h.GetMethod(ctx); want != have {
				t.Errorf("unexpected method, set: %v, have: %v", want, have)
			}
		}
	})
}

func (h *hostTester) testURI() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetURI default", func(t *testing.T) {
		if want, have := "/", h.h.GetURI(ctx); want != have {
			t.Errorf("unexpected default URI, want: %v, have: %v", want, have)
		}
	})

	tests := []struct {
		name string
		set  string
		want string
	}{
		{
			want: "/",
		},
		{
			set:  "/a b",
			want: "/a%20b",
		},
		{
			set:  "/a b?q=go+language",
			want: "/a%20b?q=go+language",
		},
		{
			set:  "/a b?q=go language",
			want: "/a%20b?q=go language",
		},
		{
			set:  "//foo",
			want: "//foo",
		},
		{
			set:  "/foo?",
			want: "/foo?",
		},
	}

	h.t.Run("SetURI", func(t *testing.T) {
		for _, tt := range tests {

			h.h.SetURI(ctx, tt.set)

			if have := h.h.GetURI(ctx); tt.want != have {
				t.Errorf("unexpected URI, set: %v, want: %v, have: %v", tt.set, tt.want, have)
			}
		}
	})
}

func (h *hostTester) testProtocolVersion() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetProtocolVersion default", func(t *testing.T) {
		if want, have := "HTTP/1.1", h.h.GetProtocolVersion(ctx); want != have {
			t.Errorf("unexpected protocol version, want: %v, have: %v", want, have)
		}
	})
}

func (h *hostTester) testRequestHeaders() {
	h.testRequestHeaderNames()
}

func (h *hostTester) testRequestHeaderNames() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetRequestHeaderNames default", func(t *testing.T) {
		if h.h.GetRequestHeaderNames(ctx) != nil {
			t.Errorf("unexpected default request header names, want: nil")
		}
	})

	h.t.Run("GetRequestHeaderNames", func(t *testing.T) {
		var want []string
		for k, vs := range testRequestHeaders {
			h.h.SetRequestHeaderValue(ctx, k, vs[0])
			want = append(want, k)
		}
		sort.Strings(want)

		have := h.h.GetRequestHeaderNames(ctx)
		sort.Strings(have)
		if !reflect.DeepEqual(want, have) {
			t.Errorf("unexpected header names, want: %v, have: %v", want, have)
		}
	})
}

func (h *hostTester) testRequestBody() {
	// All body tests require read-back
	ctx, enabled := h.newCtx(handler.FeatureBufferRequest)
	if !enabled.IsEnabled(handler.FeatureBufferRequest) {
		return
	}

	h.t.Run("RequestBodyReader default", func(t *testing.T) {
		if h.h.RequestBodyReader(ctx) == nil {
			t.Errorf("unexpected default body reader, want: != nil")
		}
	})
}

func (h *hostTester) testRequestTrailers() {
	ctx, enabled := h.newCtx(handler.FeatureTrailers)
	if !enabled.IsEnabled(handler.FeatureTrailers) {
		return
	}

	h.t.Run("GetRequestTrailerNames default", func(t *testing.T) {
		if h.h.GetRequestTrailerNames(ctx) != nil {
			t.Errorf("unexpected default trailer names, want: nil")
		}
	})
}

func (h *hostTester) testStatusCode() {
	// We can't test setting a response property without reading it back.
	// Read-back of any response property requires buffering.
	ctx, enabled := h.newCtx(handler.FeatureBufferResponse)
	if !enabled.IsEnabled(handler.FeatureBufferResponse) {
		return
	}

	h.t.Run("GetStatusCode default", func(t *testing.T) {
		if want, have := uint32(200), h.h.GetStatusCode(ctx); want != have {
			t.Errorf("unexpected default status code, want: %v, have: %v", want, have)
		}
	})
}

func (h *hostTester) testResponseHeaders() {
	h.testResponseHeaderNames()
}

func (h *hostTester) testResponseHeaderNames() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetResponseHeaderNames default", func(t *testing.T) {
		if h.h.GetResponseHeaderNames(ctx) != nil {
			t.Errorf("unexpected default response header names, want: nil")
		}
	})

	h.t.Run("GetResponseHeaderNames", func(t *testing.T) {
		var want []string
		for k, vs := range testResponseHeaders {
			h.h.SetResponseHeaderValue(ctx, k, vs[0])
			want = append(want, k)
		}
		sort.Strings(want)

		have := h.h.GetResponseHeaderNames(ctx)
		sort.Strings(have)
		if !reflect.DeepEqual(want, have) {
			t.Errorf("unexpected header names, want: %v, have: %v", want, have)
		}
	})
}

func (h *hostTester) testResponseBody() {
	// We can't test setting a response property without reading it back.
	// Read-back of any response property requires buffering.
	ctx, enabled := h.newCtx(handler.FeatureBufferResponse)
	if !enabled.IsEnabled(handler.FeatureBufferResponse) {
		return
	}

	h.t.Run("ResponseBodyReader default", func(t *testing.T) {
		if h.h.ResponseBodyReader(ctx) == nil {
			t.Errorf("unexpected default body reader, want: != nil")
		}
	})
}

func (h *hostTester) testResponseTrailers() {
	// We can't test setting a response property without reading it back.
	// Read-back of any response property requires buffering, and trailers
	// requires an additional feature
	requiredFeatures := handler.FeatureTrailers | handler.FeatureBufferResponse
	ctx, enabled := h.newCtx(requiredFeatures)
	if !enabled.IsEnabled(requiredFeatures) {
		return
	}

	h.t.Run("GetResponseTrailerNames default", func(t *testing.T) {
		if h.h.GetResponseTrailerNames(ctx) != nil {
			t.Errorf("unexpected default trailer names, want: nil")
		}
	})
}

// Note: senders are supposed to concatenate multiple fields with the same
// name on comma, except the response header Set-Cookie. That said, a lot
// of middleware don't know about this and may repeat other headers anyway.
// See https://www.rfc-editoreqHeaders.org/rfc/rfc9110#section-5.2

var (
	testRequestHeaders = map[string][]string{
		"Content-Type":    {"text/plain"},
		"Custom":          {"1"},
		"X-Forwarded-For": {"client, proxy1", "proxy2"},
		"Empty":           {""},
	}
	testResponseHeaders = map[string][]string{
		"Content-Type": {"text/plain"},
		"Custom":       {"1"},
		"Set-Cookie":   {"a=b, c=d", "e=f"},
		"Empty":        {""},
	}
)
