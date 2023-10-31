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
	"strings"
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
	ht.testSourceAddr()

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

func (h *hostTester) testSourceAddr() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetSourceAddr", func(t *testing.T) {
		addr := h.h.GetSourceAddr(ctx)
		want := "1.2.3.4:12345"
		if addr != want {
			t.Errorf("unexpected default source addr, want: %v, have: %v", want, addr)
		}
	})
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
		for _, tc := range tests {

			h.h.SetURI(ctx, tc.set)

			if have := h.h.GetURI(ctx); tc.want != have {
				t.Errorf("unexpected URI, set: %v, want: %v, have: %v", tc.set, tc.want, have)
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
	h.testGetRequestHeaderValues()
	h.testSetRequestHeaderValue()
	h.testAddRequestHeaderValue()
	h.testRemoveRequestHeaderValue()
}

func (h *hostTester) addTestRequestHeaders(ctx context.Context) {
	for k, vs := range testRequestHeaders {
		h.h.SetRequestHeaderValue(ctx, k, vs[0])
		for _, v := range vs[1:] {
			h.h.AddRequestHeaderValue(ctx, k, v)
		}
	}
}

func (h *hostTester) testRequestHeaderNames() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetRequestHeaderNames default", func(t *testing.T) {
		if h.h.GetRequestHeaderNames(ctx) != nil {
			t.Errorf("unexpected default request header names, want: nil")
		}
	})

	h.t.Run("GetRequestHeaderNames", func(t *testing.T) {
		h.addTestRequestHeaders(ctx)

		var want []string
		for k := range testRequestHeaders {
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

func (h *hostTester) testGetRequestHeaderValues() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestRequestHeaders(ctx)

	tests := []struct {
		name       string
		headerName string
		want       []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			want:       []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "X-Forwarded-For",
			want:       []string{"client, proxy1", "proxy2"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			want:       []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h.t.Run("GetRequestHeaderValues", func(t *testing.T) {
		for _, tc := range tests {
			values := h.h.GetRequestHeaderValues(ctx, tc.headerName)

			if want, have := tc.want, values; !reflect.DeepEqual(want, have) {
				t.Errorf("%s: unexpected header values, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testSetRequestHeaderValue() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestRequestHeaders(ctx)

	tests := []struct {
		name       string
		headerName string
		want       string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			want:       "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			want:       "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			want:       "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "X-Forwarded-For",
			want:       "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "X-Forwarded-For",
			want:       "",
		},
	}

	h.t.Run("SetRequestHeaderValue", func(t *testing.T) {
		for _, tc := range tests {
			h.h.SetRequestHeaderValue(ctx, tc.headerName, tc.want)

			if want, have := tc.want, strings.Join(h.h.GetRequestHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("%s: unexpected header, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testAddRequestHeaderValue() {
	tests := []struct {
		name       string
		headerName string
		value      string
		want       []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			want:       []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			want:       []string{""},
		},
		{
			name:       "existing",
			headerName: "X-Forwarded-For",
			value:      "proxy3",
			want:       []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "lowercase",
			headerName: "x-forwarded-for",
			value:      "proxy3",
			want:       []string{"client, proxy1", "proxy2", "proxy3"},
		},
		{
			name:       "existing empty",
			headerName: "X-Forwarded-For",
			want:       []string{"client, proxy1", "proxy2", ""},
		},
	}

	h.t.Run("AddRequestHeaderValue", func(t *testing.T) {
		for _, tc := range tests {
			ctx, _ := h.newCtx(0) // no features required
			h.addTestRequestHeaders(ctx)

			h.h.AddRequestHeaderValue(ctx, tc.headerName, tc.value)

			if want, have := tc.want, h.h.GetRequestHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("%s: unexpected header, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testRemoveRequestHeaderValue() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestRequestHeaders(ctx)

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

	h.t.Run("RemoveRequestHeader", func(t *testing.T) {
		for _, tc := range tests {
			h.h.RemoveRequestHeader(ctx, tc.headerName)

			if have := h.h.GetRequestHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("%s: unexpected headers: %v", tc.name, have)
			}
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
	h.testGetResponseHeaderValues()
	h.testSetResponseHeaderValue()
	h.testAddResponseHeaderValue()
	h.testRemoveResponseHeaderValue()
}

func (h *hostTester) addTestResponseHeaders(ctx context.Context) {
	for k, vs := range testResponseHeaders {
		h.h.SetResponseHeaderValue(ctx, k, vs[0])
		for _, v := range vs[1:] {
			h.h.AddResponseHeaderValue(ctx, k, v)
		}
	}
}

func (h *hostTester) testResponseHeaderNames() {
	ctx, _ := h.newCtx(0) // no features required

	h.t.Run("GetResponseHeaderNames default", func(t *testing.T) {
		if h.h.GetResponseHeaderNames(ctx) != nil {
			t.Errorf("unexpected default response header names, want: nil")
		}
	})

	h.t.Run("GetResponseHeaderNames", func(t *testing.T) {
		h.addTestResponseHeaders(ctx)

		var want []string
		for k := range testResponseHeaders {
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

func (h *hostTester) testGetResponseHeaderValues() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestResponseHeaders(ctx)

	tests := []struct {
		name       string
		headerName string
		want       []string
	}{
		{
			name:       "single value",
			headerName: "Content-Type",
			want:       []string{"text/plain"},
		},
		{
			name:       "multi-field with comma value",
			headerName: "Set-Cookie",
			want:       []string{"a=b, c=d", "e=f"},
		},
		{
			name:       "empty value",
			headerName: "Empty",
			want:       []string{""},
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h.t.Run("GetResponseHeaderValues", func(t *testing.T) {
		for _, tc := range tests {
			values := h.h.GetResponseHeaderValues(ctx, tc.headerName)

			if want, have := tc.want, values; !reflect.DeepEqual(want, have) {
				t.Errorf("%s: unexpected header values, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testSetResponseHeaderValue() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestResponseHeaders(ctx)

	tests := []struct {
		name       string
		headerName string
		want       string
	}{
		{
			name:       "non-existing",
			headerName: "custom",
			want:       "1",
		},
		{
			name:       "existing",
			headerName: "Content-Type",
			want:       "application/json",
		},
		{
			name:       "existing lowercase",
			headerName: "content-type",
			want:       "application/json",
		},
		{
			name:       "set to empty",
			headerName: "Custom",
		},
		{
			name:       "multi-field",
			headerName: "Set-Cookie",
			want:       "proxy2",
		},
		{
			name:       "set multi-field to empty",
			headerName: "Set-Cookie",
			want:       "",
		},
	}

	h.t.Run("SetResponseHeaderValue", func(t *testing.T) {
		for _, tc := range tests {
			h.h.SetResponseHeaderValue(ctx, tc.headerName, tc.want)

			if want, have := tc.want, strings.Join(h.h.GetResponseHeaderValues(ctx, tc.headerName), "|"); want != have {
				t.Errorf("%s: unexpected header, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testAddResponseHeaderValue() {
	tests := []struct {
		name       string
		headerName string
		value      string
		want       []string
	}{
		{
			name:       "non-existing",
			headerName: "new",
			value:      "1",
			want:       []string{"1"},
		},
		{
			name:       "empty",
			headerName: "new",
			want:       []string{""},
		},
		{
			name:       "existing",
			headerName: "Set-Cookie",
			value:      "g=h",
			want:       []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "lowercase",
			headerName: "set-Cookie",
			value:      "g=h",
			want:       []string{"a=b, c=d", "e=f", "g=h"},
		},
		{
			name:       "existing empty",
			headerName: "Set-Cookie",
			value:      "",
			want:       []string{"a=b, c=d", "e=f", ""},
		},
	}

	h.t.Run("AddResponseHeaderValue", func(t *testing.T) {
		for _, tc := range tests {
			ctx, _ := h.newCtx(0) // no features required
			h.addTestResponseHeaders(ctx)

			h.h.AddResponseHeaderValue(ctx, tc.headerName, tc.value)

			if want, have := tc.want, h.h.GetResponseHeaderValues(ctx, tc.headerName); !reflect.DeepEqual(want, have) {
				t.Errorf("%s: unexpected header, want: %v, have: %v", tc.name, want, have)
			}
		}
	})
}

func (h *hostTester) testRemoveResponseHeaderValue() {
	ctx, _ := h.newCtx(0) // no features required
	h.addTestResponseHeaders(ctx)

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

	h.t.Run("RemoveResponseHeader", func(t *testing.T) {
		for _, tc := range tests {
			h.h.RemoveResponseHeader(ctx, tc.headerName)

			if have := h.h.GetResponseHeaderValues(ctx, tc.headerName); len(have) > 0 {
				t.Errorf("%s: unexpected headers: %v", tc.name, have)
			}
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
