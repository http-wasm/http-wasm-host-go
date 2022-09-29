package wasm

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
)

type wasmOptions struct {
	newRuntime   NewRuntime
	moduleConfig wazero.ModuleConfig
	logger       httpwasm.Logger
}

// Option is configuration for NewWasmHandler
type Option func(*wasmOptions)

// NewRuntime returns a new wazero runtime which is called on NewMiddleware.
// the result is closed upon Middleware.Close.
type NewRuntime func(context.Context) (wazero.Runtime, error)

// DefaultRuntime implements NewRuntime by returning a wazero runtime with WASI
// host functions instantiated.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	r := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, err
	}
	return r, nil
}

// Runtime provides the wazero.Runtime and defaults to DefaultRuntime.
func Runtime(newRuntime NewRuntime) Option {
	return func(h *wasmOptions) {
		h.newRuntime = newRuntime
	}
}

// GuestConfig is the configuration used to instantiate the guest.
func GuestConfig(moduleConfig wazero.ModuleConfig) Option {
	return func(h *wasmOptions) {
		h.moduleConfig = moduleConfig
	}
}

// Logger sets the logger used by the guest when it calls "log". Defaults to
// ignore messages.
func Logger(logger httpwasm.Logger) Option {
	return func(h *wasmOptions) {
		h.logger = logger
	}
}
