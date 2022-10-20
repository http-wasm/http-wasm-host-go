package internalhandler

import (
	"bytes"
	"context"

	wazeroapi "github.com/tetratelabs/wazero/api"
)

func writeNULTerminated(
	ctx context.Context,
	mem wazeroapi.Memory,
	buf, bufLimit uint32,
	input []string,
) (countLen uint64) {
	count := uint32(len(input))
	if count == 0 {
		return
	}

	byteCount := count // NUL terminator count
	for _, s := range input {
		byteCount += uint32(len(s))
	}

	countLen = uint64(count)<<32 | uint64(byteCount)

	if byteCount > bufLimit {
		return // the guest can retry with a larger limit
	}

	// Write the NUL-terminated string to memory directly.
	s, ok := mem.Read(ctx, buf, byteCount)
	if !ok {
		panic("out of memory") // the guest passed a region outside memory.
	}

	b := bytes.NewBuffer(s)
	b.Reset()
	for _, h := range input {
		b.WriteString(h)
		b.WriteByte(0)
	}
	return
}
