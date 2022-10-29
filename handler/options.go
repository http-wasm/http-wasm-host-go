package handler

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

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

// NewNamespace returns a new wazero namespace which is called when creating a
// new handler instance, which also closes it.
type NewNamespace func(context.Context, wazero.Runtime) (wazero.Namespace, error)

// Namespace provides the wazero.Namespace and defaults to one that with
// wasi_snapshot_preview1.ModuleName instantiated.
func Namespace(newNamespace NewNamespace) Option {
	return func(h *options) {
		h.newNamespace = newNamespace
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
	newNamespace func(context.Context, wazero.Runtime) (wazero.Namespace, error)
	guestConfig  []byte
	moduleConfig wazero.ModuleConfig
	logger       api.Logger
}

// DefaultRuntime implements options.newRuntime.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	return wazero.NewRuntime(ctx), nil
}

// DefaultNamespace implements options.newNamespace.
func DefaultNamespace(ctx context.Context, r wazero.Runtime) (wazero.Namespace, error) {
	ns := r.NewNamespace(ctx)

	if _, err := wasi_snapshot_preview1.NewBuilder(r).
		Instantiate(ctx, ns); err != nil {
		_ = ns.Close(ctx)
		return nil, err
	}
	return ns, nil
}
