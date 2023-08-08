package handler

import (
	"context"
	"sync"

	wazeroapi "github.com/tetratelabs/wazero/api"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var _ Middleware = (*poolMiddleware)(nil)

type poolMiddleware struct {
	middleware
	pool sync.Pool
}

func (m *poolMiddleware) instantiateHost(ctx context.Context) error {
	if _, err := m.hostModuleBuilder().Instantiate(ctx); err != nil {
		return err
	}

	// Eagerly add one instance to the pool. Doing so helps to fail fast.
	if g, err := m.newGuest(ctx); err != nil {
		return err
	} else {
		m.pool.Put(g)
	}
	return nil
}

type poolRequestStateKey struct{}

type poolRequestState struct {
	requestState

	putPool func(x any)
	g       *guest
}

func (r *poolRequestState) Close() error {
	if g := r.g; g != nil {
		r.putPool(r.g)
		r.g = nil
	}
	return r.requestState.Close()
}

// HandleRequest implements Middleware.HandleRequest
func (m *poolMiddleware) HandleRequest(ctx context.Context) (outCtx context.Context, ctxNext handler.CtxNext, err error) {
	g, guestErr := m.getOrCreateGuest(ctx)
	if guestErr != nil {
		err = guestErr
		return
	}

	s := &poolRequestState{requestState: requestState{features: m.features}, putPool: m.pool.Put, g: g}
	defer func() {
		if ctxNext != 0 { // will call the next handler
			if closeErr := s.closeRequest(); err == nil {
				err = closeErr
			}
		} else { // guest errored or returned the response
			if closeErr := s.Close(); err == nil {
				err = closeErr
			}
		}
	}()

	outCtx = context.WithValue(ctx, requestStateKey{}, &s.requestState)
	outCtx = context.WithValue(outCtx, poolRequestStateKey{}, s)
	ctxNext, err = g.handleRequest(outCtx)
	return
}

// HandleResponse implements Middleware.HandleResponse
func (m *poolMiddleware) HandleResponse(ctx context.Context, reqCtx uint32, hostErr error) error {
	s := ctx.Value(poolRequestStateKey{}).(*poolRequestState)
	defer s.Close()
	s.afterNext = true

	return s.g.handleResponse(ctx, reqCtx, hostErr)
}

func (m *poolMiddleware) getOrCreateGuest(ctx context.Context) (*guest, error) {
	poolG := m.pool.Get()
	if poolG == nil {
		if g, createErr := m.newGuest(ctx); createErr != nil {
			return nil, createErr
		} else {
			poolG = g
		}
	}
	return poolG.(*guest), nil
}

type guest struct {
	guest            wazeroapi.Module
	handleRequestFn  wazeroapi.Function
	handleResponseFn wazeroapi.Function
}

func (m *poolMiddleware) newGuest(ctx context.Context) (*guest, error) {
	g, err := m.newModule(ctx)
	if err != nil {
		return nil, err
	}

	return &guest{
		guest:            g,
		handleRequestFn:  g.ExportedFunction(handler.FuncHandleRequest),
		handleResponseFn: g.ExportedFunction(handler.FuncHandleResponse),
	}, nil
}

// handleRequest calls the WebAssembly guest function handler.FuncHandleRequest.
func (g *guest) handleRequest(ctx context.Context) (ctxNext handler.CtxNext, err error) {
	if results, guestErr := g.handleRequestFn.Call(ctx); guestErr != nil {
		err = guestErr
	} else {
		ctxNext = handler.CtxNext(results[0])
	}
	return
}

// handleResponse calls the WebAssembly guest function handler.FuncHandleResponse.
func (g *guest) handleResponse(ctx context.Context, reqCtx uint32, err error) error {
	wasError := uint64(0)
	if err != nil {
		wasError = 1
	}
	_, err = g.handleResponseFn.Call(ctx, uint64(reqCtx), wasError)
	return err
}
