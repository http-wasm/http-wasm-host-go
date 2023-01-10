package handler

import (
	"context"

	"github.com/tetratelabs/wazero"

	"github.com/http-wasm/http-wasm-host-go/api"
)

// Option is configuration for NewMiddleware
type Option func(*options)

// NewRuntime returns a new wazero runtime which is called when creating a new
// middleware instance, which also closes it.
type NewRuntime func(context.Context) (wazero.Runtime, error)

// Runtime provides the wazero.Runtime and defaults to wazero.NewRuntime.
func Runtime(newRuntime NewRuntime) Option {
	return func(h *options) {
		h.newRuntime = newRuntime
	}
}

// GuestConfig is the configuration used to instantiate the guest.
func GuestConfig(guestConfig []byte) Option {
	return func(h *options) {
		h.guestConfig = guestConfig
	}
}

// ModuleConfig is the configuration used to instantiate the guest.
func ModuleConfig(moduleConfig wazero.ModuleConfig) Option {
	return func(h *options) {
		h.moduleConfig = moduleConfig
	}
}

// Logger sets the logger used by the guest when it calls "log". Defaults to
// api.NoopLogger.
func Logger(logger api.Logger) Option {
	return func(h *options) {
		h.logger = logger
	}
}

type options struct {
	newRuntime   func(context.Context) (wazero.Runtime, error)
	guestConfig  []byte
	moduleConfig wazero.ModuleConfig
	logger       api.Logger
}

// DefaultRuntime implements options.newRuntime.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	return wazero.NewRuntime(ctx), nil
}
