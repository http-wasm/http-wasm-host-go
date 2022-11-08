// Package handlertest implements support for testing implementations
// of HTTP handlers. This is inspired by fstest.TestFS.
package handlertest

import (
	"context"
	"errors"
	"fmt"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

// TestHost tests a handler.Host by checking default property values and
// ability to change them.
//
// To use this, pass your host and the want protocol version.
//
//	if err := handlertest.TestHost(myHost, "HTTP/1.1"); err != nil {
//		t.Fatal(err)
//	}
func TestHost(h handler.Host, newCtx func() context.Context) error {
	t := hostTester{h: h, newCtx: newCtx}

	t.checkMethod()
	t.checkURI()
	t.checkProtocolVersion()
	t.checkRequestHeaders()
	t.checkRequestBody()
	t.checkRequestTrailers()
	t.checkStatusCode()
	t.checkResponseHeaders()
	t.checkResponseBody()
	t.checkResponseTrailers()

	if len(t.errText) == 0 {
		return nil
	}
	return errors.New("TestHost found errors:\n" + string(t.errText))
}

// A hostTester holds state for running the test.
type hostTester struct {
	h       handler.Host
	newCtx  func() context.Context
	errText []byte
}

// errorf adds an error line to errText.
func (t *hostTester) errorf(format string, args ...any) {
	if len(t.errText) > 0 {
		t.errText = append(t.errText, '\n')
	}
	t.errText = append(t.errText, fmt.Sprintf(format, args...)...)
}

func (t *hostTester) checkMethod() {
	ctx := t.newCtx()

	// Check default
	if want, have := "GET", t.h.GetMethod(ctx); want != have {
		t.errorf("GetMethod: unexpected default, want: %v, have: %v", want, have)
	}

	for _, want := range []string{"POST", "OPTIONS"} {
		t.h.SetMethod(ctx, want)

		if have := t.h.GetMethod(ctx); want != have {
			t.errorf("Set/GetMethod: unexpected, set: %v, have: %v", want, have)
		}
	}
}

func (t *hostTester) checkURI() {
	ctx := t.newCtx()

	if want, have := "/", t.h.GetURI(ctx); want != have {
		t.errorf("GetURI: unexpected default, want: %v, have: %v", want, have)
	}

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

	for _, tt := range tests {
		t.h.SetURI(ctx, tt.set)

		if have := t.h.GetURI(ctx); tt.want != have {
			t.errorf("Set/GetURI: unexpected, set: %v, want: %v, have: %v", tt.set, tt.want, have)
		}
	}
}

func (t *hostTester) checkProtocolVersion() {
	ctx := t.newCtx()

	if want, have := "HTTP/1.1", t.h.GetProtocolVersion(ctx); want != have {
		t.errorf("GetProtocolVersion: unexpected, want: %v, have: %v", want, have)
	}
}

func (t *hostTester) checkRequestHeaders() {
	ctx := t.newCtx()

	if t.h.GetRequestHeaderNames(ctx) != nil {
		t.errorf("GetRequestHeaderNames: unexpected default, want: nil")
	}
}

func (t *hostTester) checkRequestBody() {
	ctx := t.newCtx()

	if t.h.RequestBodyReader(ctx) == nil {
		t.errorf("RequestBodyReader: unexpected default, want: != nil")
	}
}

func (t *hostTester) checkRequestTrailers() {
	ctx := t.newCtx()

	if t.h.GetRequestTrailerNames(ctx) != nil {
		t.errorf("GetRequestTrailerNames: unexpected default, want: nil")
	}
}

func (t *hostTester) checkStatusCode() {
	ctx := t.newCtx()

	if want, have := uint32(200), t.h.GetStatusCode(ctx); want != have {
		t.errorf("GetStatusCode: unexpected default, want: %v, have: %v", want, have)
	}
}

func (t *hostTester) checkResponseHeaders() {
	ctx := t.newCtx()

	if t.h.GetResponseHeaderNames(ctx) != nil {
		t.errorf("GetResponseHeaderNames: unexpected default, want: nil")
	}
}

func (t *hostTester) checkResponseBody() {
	ctx := t.newCtx()

	if t.h.ResponseBodyReader(ctx) == nil {
		t.errorf("ResponseBodyReader: unexpected default, want: != nil")
	}
}

func (t *hostTester) checkResponseTrailers() {
	ctx := t.newCtx()

	if t.h.GetResponseTrailerNames(ctx) != nil {
		t.errorf("GetResponseTrailerNames: unexpected default, want: nil")
	}
}
