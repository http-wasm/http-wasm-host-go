package wasm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tetratelabs/wazero"
)

// Middleware is a factory of http.Handler instances implemented in Wasm.
type Middleware interface {
	// NewHandler creates a http.Handler implemented by a WebAssembly module.
	// The returned handler will not invoke `next` when it obviates a request
	// for reasons such as an authorization failure or serving from cache.
	//
	// ## Notes
	//   - Each handler is independent, so they don't share memory.
	//   - Handlers returned are not safe for concurrent use.
	NewHandler(ctx context.Context, next http.Handler) (Handler, error)

	// Close releases resources such as any Wasm modules, compiled code, and
	// the runtime.
	Close(context.Context) error
}

func NewMiddleware(ctx context.Context, guest []byte, options ...Option) (Middleware, error) {
	o := &wasmOptions{
		newRuntime:   DefaultRuntime,
		moduleConfig: wazero.NewModuleConfig(),
		logger:       func(msg string) {},
	}
	for _, option := range options {
		option(o)
	}

	r, err := o.newRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("wasm: error creating runtime: %w", err)
	}

	mw := &wasmMiddleware{
		runtime: r,
		config:  o.moduleConfig,
	}

	if mw.host, err = compileHost(ctx, r, o.logger); err != nil {
		_ = mw.Close(ctx)
		return nil, err
	}

	if mw.guest, err = compileGuest(ctx, r, guest); err != nil {
		_ = mw.Close(ctx)
		return nil, err
	}

	return mw, nil
}

var _ Middleware = &wasmMiddleware{}

type wasmMiddleware struct {
	runtime     wazero.Runtime
	guest, host wazero.CompiledModule
	config      wazero.ModuleConfig
}

// NewHandler implements Middleware.NewHandler
func (w *wasmMiddleware) NewHandler(ctx context.Context, next http.Handler) (Handler, error) {
	h := &wasmHandler{ns: w.runtime.NewNamespace(ctx), next: next}

	// Note: host modules don't use configuration
	_, err := h.ns.InstantiateModule(ctx, w.host, wazero.NewModuleConfig())
	if err != nil {
		_ = h.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating host: %w", err)
	}

	if h.guest, err = h.ns.InstantiateModule(ctx, w.guest, w.config); err != nil {
		_ = h.Close(ctx)
		return nil, fmt.Errorf("wasm: error instantiating guest: %w", err)
	}

	return h, nil
}

// Close implements Middleware.Close
func (w *wasmMiddleware) Close(ctx context.Context) error {
	// We don't have to close the guest and host as the runtime will close them.
	return w.runtime.Close(ctx)
}
