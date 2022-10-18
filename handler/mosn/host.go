package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/header"
	"mosn.io/pkg/protocol/http"

	"github.com/http-wasm/http-wasm-host-go/api/handler"
)

var _ handler.Host = host{}

type host struct{}

// EnableFeatures implements the same method as documented on handler.Host.
func (host) EnableFeatures(ctx context.Context, features handler.Features) handler.Features {
	// Remove trailers until it is supported. See mosn/mosn#2145
	features = features &^ handler.FeatureTrailers
	if s, ok := ctx.Value(filterKey{}).(*filter); ok {
		s.enableFeatures(features)
	}
	return features
}

func (host) GetMethod(ctx context.Context) (method string) {
	return mustGetString(ctx, types.VarMethod)
}

func (host) SetMethod(ctx context.Context, method string) {
	mustSetString(ctx, types.VarMethod, method)
}

func (host) GetProtocolVersion(ctx context.Context) string {
	p := mustGetString(ctx, types.VarProtocol)
	switch p {
	case "Http1":
		return "HTTP/1.1"
	case "Http2":
		return "HTTP/2.0"
	}
	return p
}

func (host) GetRequestHeaderNames(ctx context.Context) (names []string) {
	return getHeaderNames(filterFromContext(ctx).reqHeaders)
}

func (host) GetRequestHeader(ctx context.Context, name string) (string, bool) {
	return getHeader(filterFromContext(ctx).reqHeaders, name)
}

func (host) GetRequestHeaders(ctx context.Context, name string) []string {
	return getHeaders(filterFromContext(ctx).reqHeaders, name)
}

func (host) SetRequestHeader(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	f.reqHeaders = setHeader(f.reqHeaders, name, value)
}

func (host) AddRequestHeader(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	f.respHeaders = addHeader(f.respHeaders, name, value)
}

func (host) RemoveRequestHeader(ctx context.Context, name string) {
	f := filterFromContext(ctx)
	f.respHeaders.Del(name)
}

func (host) RequestBodyReader(ctx context.Context) io.ReadCloser {
	b := filterFromContext(ctx).reqBody.Bytes()
	return io.NopCloser(bytes.NewReader(b))
}

func (host) RequestBodyWriter(ctx context.Context) io.Writer {
	f := filterFromContext(ctx)
	f.reqBody.Reset()
	return writerFunc(f.WriteRequestBody)
}

func (host) GetRequestTrailerNames(ctx context.Context) (names []string) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) GetRequestTrailer(ctx context.Context, name string) (value string, ok bool) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) GetRequestTrailers(ctx context.Context, name string) (values []string) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) AddRequestTrailer(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
}

func (host) RemoveRequestTrailer(ctx context.Context, name string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
}

func (host) SetRequestTrailer(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
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

func (host) GetResponseHeaderNames(ctx context.Context) (names []string) {
	return getHeaderNames(filterFromContext(ctx).respHeaders)
}

func (host) GetResponseHeader(ctx context.Context, name string) (string, bool) {
	return getHeader(filterFromContext(ctx).respHeaders, name)
}

func (host) GetResponseHeaders(ctx context.Context, name string) []string {
	return getHeaders(filterFromContext(ctx).respHeaders, name)
}

func (host) SetResponseHeader(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	f.respHeaders = setHeader(f.respHeaders, name, value)
}

func (host) AddResponseHeader(ctx context.Context, name, value string) {
	f := filterFromContext(ctx)
	f.respHeaders = addHeader(f.respHeaders, name, value)
}

func (host) RemoveResponseHeader(ctx context.Context, name string) {
	f := filterFromContext(ctx)
	f.respHeaders.Del(name)
}

func (host) ResponseBodyReader(ctx context.Context) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(filterFromContext(ctx).respBody))
}

func (host) ResponseBodyWriter(ctx context.Context) io.Writer {
	f := filterFromContext(ctx)
	f.respBody = nil // reset
	return writerFunc(f.WriteResponseBody)
}

func (host) GetResponseTrailerNames(ctx context.Context) (names []string) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) GetResponseTrailer(ctx context.Context, name string) (value string, ok bool) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) GetResponseTrailers(ctx context.Context, name string) (values []string) {
	return // no-op because trailers are unsupported: mosn/mosn#2145
}

func (host) SetResponseTrailer(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
}

func (host) AddResponseTrailer(ctx context.Context, name, value string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
}

func (host) RemoveResponseTrailer(ctx context.Context, name string) {
	// panic because the user should know that trailers are not supported.
	panic("trailers unsupported: mosn/mosn#2145")
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

func getHeaderNames(headers api.HeaderMap) (names []string) {
	if headers == nil {
		return
	}

	// headers.Range can return the same name multiple times, so de-dupe
	var headerNames map[string]struct{}

	headers.Range(func(key, value string) bool {
		if headerNames == nil {
			headerNames = map[string]struct{}{key: {}}
		} else if _, ok := headerNames[key]; ok {
			return true // dupe
		} else {
			headerNames[key] = struct{}{}
		}
		names = append(names, key)
		return true
	})
	return
}

func getHeader(headers api.HeaderMap, name string) (string, bool) {
	if headers == nil {
		return "", false
	}
	return headers.Get(name)
}

func getHeaders(headers api.HeaderMap, name string) (values []string) {
	if headers == nil {
		return
	}
	headers.Range(func(key, value string) bool {
		if key == name {
			values = append(values, value)
		}
		return true
	})
	return
}

func setHeader(headers api.HeaderMap, name string, value string) api.HeaderMap {
	if headers == nil {
		return header.CommonHeader(map[string]string{name: value})
	}
	headers.Set(name, value)
	return headers
}

func addHeader(headers api.HeaderMap, name string, value string) api.HeaderMap {
	if headers == nil {
		return header.CommonHeader(map[string]string{name: value})
	}
	headers.Add(name, value)
	return headers
}
