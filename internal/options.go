package internal

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/http-wasm/http-wasm-host-go/api"
)

type WazeroOptions struct {
	NewRuntime   func(context.Context) (wazero.Runtime, error)
	NewNamespace func(context.Context, wazero.Runtime) (wazero.Namespace, error)
	GuestConfig  []byte
	ModuleConfig wazero.ModuleConfig
	Logger       api.LogFunc
}

// DefaultRuntime implements WazeroOptions.NewRuntime.
func DefaultRuntime(ctx context.Context) (wazero.Runtime, error) {
	return wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter()), nil
}

// DefaultNamespace implements WazeroOptions.NewNamespace.
func DefaultNamespace(ctx context.Context, r wazero.Runtime) (wazero.Namespace, error) {
	ns := r.NewNamespace(ctx)

	if _, err := wasi_snapshot_preview1.NewBuilder(r).
		Instantiate(ctx, ns); err != nil {
		_ = ns.Close(ctx)
		return nil, err
	}
	return ns, nil
}
