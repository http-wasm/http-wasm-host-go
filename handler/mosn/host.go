package mosn

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/header"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
	internalhandler "github.com/http-wasm/http-wasm-host-go/internal/handler"
)

var _ handler.Host = (*host)(nil)

type host struct {
}

func (host) GetRequestBody(ctx context.Context) []byte {
	return filterFromContext(ctx).reqBody.Bytes()
}

func (host) SetRequestBody(ctx context.Context, body []byte) {
	buf := filterFromContext(ctx).reqBody
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
	// TODO(anuraaga): There is also variable.GetProtocolResource(ctx, api.URI), unclear if they must be kept in sync.
	p, _ := variable.GetString(ctx, types.VarPath)
	q, qErr := variable.GetString(ctx, types.VarQueryString)
	if qErr != nil {
		// No query, so an error is returned.
		return p
	}
	return fmt.Sprintf("%s?%s", p, q)
}

func (host) SetURI(ctx context.Context, uri string) {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		panic(err)
	}
	_ = variable.SetString(ctx, types.VarPath, u.Path)
	if len(u.RawQuery) > 0 || u.ForceQuery {
		_ = variable.SetString(ctx, types.VarQueryString, u.RawQuery)
	}
}

func (host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	return filterFromContext(ctx).reqHeaders.Get(name)
}

func (host) SetResponseHeader(ctx context.Context, name, value string) {
	hdrs := filterFromContext(ctx).respHeaders
	if hdrs == nil {
		hdrs = header.CommonHeader(make(map[string]string))
		filterFromContext(ctx).respHeaders = hdrs
	}
	hdrs.Set(name, value)
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
	if status, ok := f.respHeaders.Get(":status"); ok {
		if code, err := strconv.Atoi(status); err == nil {
			return uint32(code)
		}
	}
	return 0
}

func (host) SetStatusCode(ctx context.Context, statusCode uint32) {
	filterFromContext(ctx).respStatus = int(statusCode)
}

func (host) GetResponseBody(ctx context.Context) []byte {
	return filterFromContext(ctx).respBody
}

func (host) SetResponseBody(ctx context.Context, body []byte) {
	filterFromContext(ctx).respBody = body
}
