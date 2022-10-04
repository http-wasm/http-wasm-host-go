package httpwasm

import (
	"context"

	"github.com/tetratelabs/wazero"

	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/internal"
)

// Option is configuration for NewWasmHandler
type Option func(*internal.WazeroOptions)

// NewRuntime returns a new wazero runtime which is called when creating a new
// middleware instance, which also closes it.
type NewRuntime func(context.Context) (wazero.Runtime, error)

// Runtime provides the wazero.Runtime and defaults to wazero.NewRuntime.
func Runtime(newRuntime NewRuntime) Option {
	return func(h *internal.WazeroOptions) {
		h.NewRuntime = newRuntime
	}
}

// NewNamespace returns a new wazero namespace which is called when creating a
// new handler instance, which also closes it.
type NewNamespace func(context.Context, wazero.Runtime) (wazero.Namespace, error)

// Namespace provides the wazero.Namespace and defaults to one that with
// wasi_snapshot_preview1.ModuleName instantiated.
func Namespace(newNamespace NewNamespace) Option {
	return func(h *internal.WazeroOptions) {
		h.NewNamespace = newNamespace
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
