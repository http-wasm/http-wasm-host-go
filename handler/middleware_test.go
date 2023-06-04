package handler

import (
	"context"
	_ "embed"
	"reflect"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

var testCtx = context.Background()

func TestNewMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		guest         []byte
		expectedError string
	}{
		{
			name:  "ok",
			guest: test.BinE2EProtocolVersion,
		},
		{
			name:  "panic on _start",
			guest: test.BinErrorPanicOnStart,
			expectedError: `wasm: error instantiating guest: module[1] function[_start] failed: wasm error: unreachable
wasm stack trace:
	panic_on_start.main()`,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw, err := NewMiddleware(testCtx, tc.guest, handler.UnimplementedHost{})
			requireEqualError(t, err, tc.expectedError)
			if mw != nil {
				mw.Close(testCtx)
			}
		})
	}
}

func TestMiddlewareHandleRequest_Error(t *testing.T) {
	tests := []struct {
		name          string
		guest         []byte
		expectedError string
	}{
		{
			name:  "panic",
			guest: test.BinErrorPanicOnHandleRequest,
			expectedError: `wasm error: unreachable
wasm stack trace:
	panic_on_handle_request.handle_request() i64`,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			mw, err := NewMiddleware(testCtx, tc.guest, handler.UnimplementedHost{})
			if err != nil {
				t.Fatal(err)
			}
			defer mw.Close(testCtx)

			_, _, err = mw.HandleRequest(testCtx)
			requireEqualError(t, err, tc.expectedError)
		})
	}
}

func TestMiddlewareHandleResponse_Error(t *testing.T) {
	tests := []struct {
		name          string
		guest         []byte
		expectedError string
	}{
		{
			name:  "panic",
			guest: test.BinErrorPanicOnHandleResponse,
			expectedError: `wasm error: unreachable
wasm stack trace:
	panic_on_handle_response.handle_response(i32,i32)`,
		},
		{
			name:  "set_header_value request",
			guest: test.BinErrorSetRequestHeaderAfterNext,
			expectedError: `can't set request header after next handler (recovered by wazero)
wasm stack trace:
	http_handler.set_header_value(i32,i32,i32,i32,i32)
	set_request_header_after_next.handle_response(i32,i32)`,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			mw, err := NewMiddleware(testCtx, tc.guest, handler.UnimplementedHost{})
			if err != nil {
				t.Fatal(err)
			}
			defer mw.Close(testCtx)

			// We don't expect an error on the request path
			ctx, ctxNext, err := mw.HandleRequest(testCtx)
			requireHandleRequest(t, mw, ctxNext, err, 0)

			// We do expect an error on the response path
			err = mw.HandleResponse(ctx, 0, nil)
			requireEqualError(t, err, tc.expectedError)
		})
	}
}

func TestMiddlewareResponseUsesRequestModule(t *testing.T) {
	mw, err := NewMiddleware(testCtx, test.BinE2EHandleResponse, handler.UnimplementedHost{})
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	// A new guest module has initial state, so its value should be 42
	r1Ctx, ctxNext, err := mw.HandleRequest(testCtx)
	requireHandleRequest(t, mw, ctxNext, err, 42)

	// The first guest shouldn't return to the pool until HandleResponse, so
	// the second simultaneous call will get a new guest.
	r2Ctx, ctxNext2, err := mw.HandleRequest(testCtx)
	requireHandleRequest(t, mw, ctxNext2, err, 42)

	// Return the first request to the pool
	if err = mw.HandleResponse(r1Ctx, uint32(ctxNext>>32), nil); err != nil {
		t.Fatal(err)
	}
	requireGlobals(t, mw, 43)

	// The next request should re-use the returned module
	r3Ctx, ctxNext3, err := mw.HandleRequest(testCtx)
	requireHandleRequest(t, mw, ctxNext3, err, 43)
	if err = mw.HandleResponse(r3Ctx, uint32(ctxNext3>>32), nil); err != nil {
		t.Fatal(err)
	}
	requireGlobals(t, mw, 44)

	// Return the second request to the pool
	if err = mw.HandleResponse(r2Ctx, uint32(ctxNext2>>32), nil); err != nil {
		t.Fatal(err)
	}
	requireGlobals(t, mw, 44, 43)
}

func requireGlobals(t *testing.T, mw Middleware, wantGlobals ...uint64) {
	t.Helper()
	if want, have := wantGlobals, getGlobalVals(mw); !reflect.DeepEqual(want, have) {
		t.Errorf("unexpected globals, want: %v, have: %v", want, have)
	}
}

func getGlobalVals(mw Middleware) []uint64 {
	pool := mw.(*middleware).pool
	var guests []*guest
	var globals []uint64

	// Take all guests out of the pool
	for {
		if g, ok := pool.Get().(*guest); ok {
			guests = append(guests, g)
			continue
		}
		break
	}

	for _, g := range guests {
		v := g.guest.ExportedGlobal("reqCtx").Get()
		globals = append(globals, v)
		pool.Put(g)
	}

	return globals
}

func requireHandleRequest(t *testing.T, mw Middleware, ctxNext handler.CtxNext, err error, expectedCtx handler.CtxNext) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := expectedCtx, ctxNext>>32; want != have {
		t.Errorf("unexpected ctx, want: %d, have: %d", want, have)
	}
	if mw.(*middleware).pool.Get() != nil {
		t.Error("expected handler to not return guest to the pool")
	}
}

func requireEqualError(t *testing.T, err error, expectedError string) {
	if err != nil {
		if want, have := expectedError, err.Error(); want != have {
			t.Fatalf("unexpected error: want %v, have %v", want, have)
		}
	} else if want := expectedError; want != "" {
		t.Fatalf("expected error %v", want)
	}
}
