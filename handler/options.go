package handler

import (
	"github.com/tetratelabs/wazero"

	"github.com/http-wasm/http-wasm-host-go/api"
)

// Option is configuration for NewMiddleware
type Option func(*options)

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
	guestConfig   []byte
	moduleConfig  wazero.ModuleConfig
	logger        api.Logger
	runtimeConfig wazero.RuntimeConfig
}
