package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"

	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/header"
	"mosn.io/pkg/protocol/http"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var _ handler.Host = host{}

type host struct{}

func (host) EnableFeatures(ctx context.Context, features handler.Features) {
	f := ctx.Value(filterKey{}).(*filter)
	f.enableFeatures(features)
}

func (h host) GetMethod(ctx context.Context) (method string) {
	return mustGetString(ctx, types.VarMethod)
}

func (h host) SetMethod(ctx context.Context, method string) {
	mustSetString(ctx, types.VarMethod, method)
}

func (h host) GetProtocolVersion(ctx context.Context) string {
	p := mustGetString(ctx, types.VarProtocol)
	switch p {
	case "Http1":
		return "HTTP/1.1"
	case "Http2":
		return "HTTP/2.0"
	}
	return p
}

func (h host) GetRequestHeaderNames(ctx context.Context) (names []string) {
	filterFromContext(ctx).reqHeaders.Range(func(key, value string) bool {
		names = append(names, key)
		return true
	})
	return
}

func (host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	return filterFromContext(ctx).reqHeaders.Get(name)
}

func (host) SetRequestHeader(ctx context.Context, name, value string) {
	req := filterFromContext(ctx).reqHeaders.(mosnhttp.RequestHeader)
	req.Set(name, value)
}

func (host) ReadRequestBody(ctx context.Context) io.ReadCloser {
	b := filterFromContext(ctx).reqBody.Bytes()
	return io.NopCloser(bytes.NewReader(b))
}

func (host) SetRequestBody(ctx context.Context, body []byte) {
	buf := filterFromContext(ctx).reqBody
	buf.Reset()
	_ = buf.Append(body)
}

func (host) GetURI(ctx context.Context) string {
	// TODO(anuraaga): There is also variable.GetProtocolResource(ctx, api.URI), unclear if they must be kept in sync.
	p := mustGetString(ctx, types.VarPath)
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
	mustSetString(ctx, types.VarPath, u.Path)
	if len(u.RawQuery) > 0 || u.ForceQuery {
		mustSetString(ctx, types.VarQueryString, u.RawQuery)
	}
}

func (host) Next(ctx context.Context) {
	f := filterFromContext(ctx)
	f.nextCalled = true

	// The handling of a request is split into two functions, OnReceive and Append in mosn.
	// Invoking the next handler means we need to finish OnReceive and come back in Append.

	// Resume execution of OnReceive which is currently waiting on this channel.
	f.ch <- nil

	// Wait for Append to resume execution of Next when it signals this channel.
	<-f.ch
}

func (host) GetStatusCode(ctx context.Context) uint32 {
	f := filterFromContext(ctx)
	if resp, ok := f.respHeaders.(http.ResponseHeader); ok {
		return uint32(resp.StatusCode())
	} else {
		return uint32(f.statusCode)
	}
}

func (host) SetStatusCode(ctx context.Context, statusCode uint32) {
	f := filterFromContext(ctx)
	if resp, ok := f.respHeaders.(http.ResponseHeader); ok {
		resp.SetStatusCode(int(statusCode))
	} else {
		f.statusCode = int(statusCode)
	}
}

func (h host) GetResponseHeaderNames(ctx context.Context) (names []string) {
	filterFromContext(ctx).respHeaders.Range(func(key, value string) bool {
		names = append(names, key)
		return true
	})
	return
}

func (host) GetResponseHeader(ctx context.Context, name string) (string, bool) {
	return filterFromContext(ctx).respHeaders.Get(name)
}

func (host) SetResponseHeader(ctx context.Context, name, value string) {
	hdrs := filterFromContext(ctx).respHeaders
	if hdrs == nil {
		hdrs = header.CommonHeader(make(map[string]string))
		filterFromContext(ctx).respHeaders = hdrs
	}
	hdrs.Set(name, value)
}

func (host) ReadResponseBody(ctx context.Context) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(filterFromContext(ctx).respBody))
}

func (host) SetResponseBody(ctx context.Context, body []byte) {
	filterFromContext(ctx).respBody = body
}

func mustGetString(ctx context.Context, name string) string {
	if s, err := variable.GetString(ctx, name); err != nil {
		panic(err)
	} else {
		return s
	}
}

func mustSetString(ctx context.Context, name, value string) {
	if err := variable.SetString(ctx, name, value); err != nil {
		panic(err)
	}
}
