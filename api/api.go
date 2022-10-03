package api

import "context"

// LogFunc writes a message to the host console.
type LogFunc func(ctx context.Context, msg string)

type Closer interface {
	// Close releases resources such as any Wasm modules, compiled code, and
	// the runtime.
	Close(context.Context) error
}
