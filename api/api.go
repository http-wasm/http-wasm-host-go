package api

import (
	"context"
)

// LogFunc logs a message to the host's logs.
type LogFunc func(context.Context, string)

type Closer interface {
	// Close releases resources such as any Wasm modules, compiled code, and
	// the runtime.
	Close(context.Context) error
}
