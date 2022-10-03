package httpwasm

import (
	"context"

	"github.com/tetratelabs/wazero"

	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/internal"
)

// Option is configuration for NewWasmHandler
type Option func(*internal.WazeroOptions)

// NewRuntime returns a new wazero runtime which is called on NewMiddleware.
// the result is closed upon Middleware.Close.
type NewRuntime func(context.Context) (wazero.Runtime, error)

// Runtime provides the wazero.Runtime and defaults to DefaultRuntime.
func Runtime(newRuntime NewRuntime) Option {
	return func(h *internal.WazeroOptions) {
		h.NewRuntime = newRuntime
	}
}

// GuestConfig is the configuration used to instantiate the guest.
func GuestConfig(moduleConfig wazero.ModuleConfig) Option {
	return func(h *internal.WazeroOptions) {
		h.ModuleConfig = moduleConfig
	}
}

// Logger sets the logger used by the guest when it calls "log". Defaults to
// ignore messages.
func Logger(logger api.LogFunc) Option {
	return func(h *internal.WazeroOptions) {
		h.Logger = logger
	}
}
