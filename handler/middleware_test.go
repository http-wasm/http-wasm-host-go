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

func Test_MiddlewareResponseUsesRequestModule(t *testing.T) {
	mw, err := NewMiddleware(testCtx, test.BinE2EHandleResponse, handler.UnimplementedHost{})
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(testCtx)

	// A new guest module has initial state, so its value should be 42
	r1Ctx, ctxNext, err := mw.HandleRequest(testCtx)
	expectHandleRequest(t, mw, ctxNext, err, 42)

	// The first guest shouldn't return to the pool until HandleResponse, so
	// the second simultaneous call will get a new guest.
	r2Ctx, ctxNext2, err := mw.HandleRequest(testCtx)
	expectHandleRequest(t, mw, ctxNext2, err, 42)

	// Return the first request to the pool
	if err = mw.HandleResponse(r1Ctx, uint32(ctxNext>>32), nil); err != nil {
		t.Fatal(err)
	}
	expectGlobals(t, mw, 43)

	// The next request should re-use the returned module
	r3Ctx, ctxNext3, err := mw.HandleRequest(testCtx)
	expectHandleRequest(t, mw, ctxNext3, err, 43)
	if err = mw.HandleResponse(r3Ctx, uint32(ctxNext3>>32), nil); err != nil {
		t.Fatal(err)
	}
	expectGlobals(t, mw, 44)

	// Return the second request to the pool
	if err = mw.HandleResponse(r2Ctx, uint32(ctxNext2>>32), nil); err != nil {
		t.Fatal(err)
	}
	expectGlobals(t, mw, 44, 43)
}

func expectGlobals(t *testing.T, mw Middleware, wantGlobals ...uint64) {
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

func expectHandleRequest(t *testing.T, mw Middleware, ctxNext handler.CtxNext, err error, expectedCtx handler.CtxNext) {
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
