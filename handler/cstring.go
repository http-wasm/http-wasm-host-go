package handler

import (
	"bytes"
	"context"

	wazeroapi "github.com/tetratelabs/wazero/api"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

func writeNULTerminated(
	ctx context.Context,
	mem wazeroapi.Memory,
	buf uint32, bufLimit handler.BufLimit,
	input []string,
) (countLen handler.CountLen) {
	count := uint32(len(input))
	if count == 0 {
		return
	}

	byteCount := count // NUL terminator count
	for _, s := range input {
		byteCount += uint32(len(s))
	}

	countLen = handler.CountLen(count)<<32 | handler.CountLen(byteCount)

	if byteCount > bufLimit {
		return // the guest can retry with a larger limit
	}

	// Write the NUL-terminated string to memory directly.
	s, ok := mem.Read(buf, byteCount)
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
