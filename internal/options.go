package internal

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/http-wasm/http-wasm-host-go/api"
)

type WazeroOptions struct {
	NewRuntime   func(context.Context) (wazero.Runtime, error)
	ModuleConfig wazero.ModuleConfig
	Logger       api.LogFunc
}

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
