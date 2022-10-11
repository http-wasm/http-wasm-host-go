package httpwasm

import (
	"context"
	"strconv"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

var _ handler.Host = (*host)(nil)

type host struct {
}

func (host) GetRequestBody(ctx context.Context) []byte {
	f := filterFromContext(ctx)
	if d := f.receiverFilterHandler.GetRequestData(); d != nil {
		return d.Bytes()
	}
	return nil
}

func (host) SetRequestBody(ctx context.Context, body []byte) {
	f := filterFromContext(ctx)
	buf := f.receiverFilterHandler.GetRequestData()
	buf.Reset()
	_ = buf.Append(body)
}

func (host) EnableFeatures(ctx context.Context, features handler.Features) handler.Features {
	if f, ok := ctx.Value(filterKey{}).(*filter); ok {
		f.enableFeatures(features)
		return f.features
	} else if i, ok := ctx.Value(internalhandler.InitStateKey{}).(*internalhandler.InitState); ok {
		i.Features = i.Features.WithEnabled(features)
		return i.Features
	} else {
		panic("unexpected context state")
	}
}

func (host) GetURI(ctx context.Context) string {
	if p, ok := filterFromContext(ctx).receiverFilterHandler.GetRequestHeaders().Get(":path"); ok {
		return p
	}
	return ""
}

func (host) SetURI(ctx context.Context, path string) {
	hdrs := filterFromContext(ctx).receiverFilterHandler.GetRequestHeaders()
	hdrs.Set(":path", path)
	filterFromContext(ctx).receiverFilterHandler.SetRequestHeaders(hdrs)
}

func (host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	return filterFromContext(ctx).receiverFilterHandler.GetRequestHeaders().Get(name)
}

func (host) SetResponseHeader(ctx context.Context, name, value string) {
	hdrs := filterFromContext(ctx).senderFilterHandler.GetResponseHeaders()
	hdrs.Set(name, value)
	filterFromContext(ctx).senderFilterHandler.SetResponseHeaders(hdrs)
}

func (host) Next(ctx context.Context) {
	f := filterFromContext(ctx)
	f.nextCalled = true
	f.ch <- nil

	<-f.ch
}

func (host) GetStatusCode(ctx context.Context) uint32 {
	f := filterFromContext(ctx)
	if f.respStatus != 0 {
		return uint32(f.respStatus)
	}
	if status, ok := f.senderFilterHandler.GetResponseHeaders().Get(":status"); ok {
		if code, err := strconv.Atoi(status); err == nil {
			return uint32(code)
		}
	}
	return 0
}

func (host) SetStatusCode(ctx context.Context, statusCode uint32) {
	f := filterFromContext(ctx)
	f.respStatus = int(statusCode)
}

func (host) GetResponseBody(ctx context.Context) []byte {
	f := filterFromContext(ctx)
	if f.respBody == nil {
		if d := f.senderFilterHandler.GetResponseData(); d != nil {
			return d.Bytes()
		}
	}
	return f.respBody
}

func (host) SetResponseBody(ctx context.Context, body []byte) {
	f := filterFromContext(ctx)
	f.respBody = body
}
