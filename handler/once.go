package handler

import (
	"context"

	wazeroapi "github.com/tetratelabs/wazero/api"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var _ Middleware = (*onceMiddleware)(nil)

type onceMiddleware struct {
	middleware
}

type onceRequestStateKey struct{}

type onceRequestState struct {
	requestState

	g wazeroapi.Module

	awaitingResponse         chan handler.CtxNext
	responseReady, guestDone chan error
}

func (r *onceRequestState) Close() error {
	if g := r.g; g != nil {
		_ = g.Close(context.Background())
	}
	return r.requestState.Close()
}

// HandleRequest implements Middleware.HandleRequest
func (m *onceMiddleware) HandleRequest(ctx context.Context) (context.Context, handler.CtxNext, error) {
	s := &onceRequestState{
		requestState:     requestState{features: m.features},
		awaitingResponse: make(chan handler.CtxNext, 1),
		responseReady:    make(chan error, 1),
		guestDone:        make(chan error, 1),
	}
	ctx = context.WithValue(ctx, requestStateKey{}, &s.requestState)
	ctx = context.WithValue(ctx, onceRequestStateKey{}, s)

	// instantiate the module in a new goroutine because it will block on
	// awaitResponse. This goroutine might outlive HandleRequest.
	go func() {
		g, err := m.newModule(ctx)
		s.g = g
		s.guestDone <- err
	}()

	select {
	case <-ctx.Done(): // ensure any context timeout applies
		_ = s.Close()
		return nil, 0, ctx.Err()
	case err := <-s.guestDone:
		_ = s.Close()
		return nil, 0, err
	case ctxNext := <-s.awaitingResponse:
		if ctxNext != 0 { // will call the next handler
			return ctx, ctxNext, s.closeRequest()
		} else { // guest returned the response
			return nil, 0, s.Close()
		}
	}
}

// HandleResponse implements Middleware.HandleResponse
func (m *onceMiddleware) HandleResponse(ctx context.Context, _ uint32, hostErr error) error {
	s := ctx.Value(onceRequestStateKey{}).(*onceRequestState)

	s.afterNext = true
	s.responseReady <- hostErr // unblock awaitResponse

	// Wait until the goroutine completes.
	select {
	case <-ctx.Done(): // ensure any context timeout applies
		_ = s.Close()
		return nil
	case err := <-s.guestDone: // block until the guest completes
		_ = s.Close()
		return err
	}
}

// awaitResponse implements the WebAssembly host function
// handler.FuncAwaitResponse.
func awaitResponse(ctx context.Context, stack []uint64) {
	s := ctx.Value(onceRequestStateKey{}).(*onceRequestState)
	s.awaitingResponse <- handler.CtxNext(stack[0])
	hostErr := <-s.responseReady
	if hostErr != nil {
		stack[0] = 1
	} else {
		stack[0] = 0
	}
}

func (m *onceMiddleware) instantiateHost(ctx context.Context) error {
	_, err := m.hostModuleBuilder().
		NewFunctionBuilder().
		WithGoFunction(wazeroapi.GoFunc(awaitResponse), []wazeroapi.ValueType{i64}, []wazeroapi.ValueType{i32}).
		WithParameterNames().Export(handler.FuncAwaitResponse).
		Instantiate(ctx)
	return err
}
