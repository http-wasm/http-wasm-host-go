package wasm

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

func newRequestHeader() mosnhttp.RequestHeader {
	return mosnhttp.RequestHeader{RequestHeader: &fasthttp.RequestHeader{}}
}

func newResponseHeader() mosnhttp.ResponseHeader {
	return mosnhttp.ResponseHeader{ResponseHeader: &fasthttp.ResponseHeader{}}
}

func Test_host_GetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: newRequestHeader()})

			if err := variable.SetString(ctx, types.VarMethod, tc); err != nil {
				t.Fatal(err)
			}

			if want, have := tc, h.GetMethod(ctx); want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetMethod(t *testing.T) {
	tests := []string{"GET", "POST", "OPTIONS"}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc, func(t *testing.T) {
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: newRequestHeader()})

			h.SetMethod(ctx, tc)

			m, err := variable.GetString(ctx, types.VarMethod)
			if err != nil {
				t.Fatal(err)
			}

			if want, have := tc, m; want != have {
				t.Errorf("unexpected method, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetURI(t *testing.T) {
	tests := []struct {
		name        string
		path, query string
		expected    string
	}{
		{
			name:     "slash",
			path:     "/",
			expected: "/",
		},
		{
			name:     "space",
			path:     "/ ",
			expected: "/ ",
		},
		{
			name:     "space query",
			path:     "/ ",
			query:    "q=go+language",
			expected: "/ ?q=go+language",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: newRequestHeader()})

			if err := variable.SetString(ctx, types.VarPath, tc.path); err != nil {
				t.Fatal(err)
			}
			if tc.query != "" {
				if err := variable.SetString(ctx, types.VarQueryString, tc.query); err != nil {
					t.Fatal(err)
				}
			}

			if want, have := tc.expected, h.GetURI(ctx); want != have {
				t.Errorf("unexpected uri, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetURI(t *testing.T) {
	tests := []struct {
		name                              string
		uri                               string
		expectedPath, expectedQueryString string
	}{
		{
			name:         "empty",
			uri:          "",
			expectedPath: "/",
		},
		{
			name:         "slash",
			uri:          "/",
			expectedPath: "/",
		},
		{
			name:         "space",
			uri:          "/ ",
			expectedPath: "/ ",
		},
		{
			name:                "space query",
			uri:                 "/ ?q=go+language",
			expectedPath:        "/ ",
			expectedQueryString: "q=go+language",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			ctx := variable.NewVariableContext(context.Background())
			// Ensure there's an existing entry for query string.
			variable.SetString(ctx, types.VarQueryString, "")

			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: newRequestHeader()})

			h.SetURI(ctx, tc.uri)

			if want, have := mustGetString(ctx, types.VarPath), mustGetString(ctx, types.VarPathOriginal); want != have {
				t.Errorf("expected paths to match, want: %v, have: %v", want, have)
			}

			if have, err := variable.GetString(ctx, types.VarPath); err != nil {
				t.Fatal(err)
			} else if want := tc.expectedPath; want != have {
				t.Errorf("unexpected path, want: %v, have: %v", want, have)
			}

			if have, err := variable.GetString(ctx, types.VarQueryString); err != nil {
				t.Fatal(err)
			} else if want := tc.expectedQueryString; want != have {
				t.Errorf("unexpected query string, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetRequestHeaderNames(t *testing.T) {
	reqHeaders := newRequestHeader()
	addTestHeaders(reqHeaders)
	ctx := variable.NewVariableContext(context.Background())
	ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: reqHeaders})

	want := []string{"Content-Type", "Vary", "Empty"}
	have := host{}.GetRequestHeaderNames(ctx)
	sort.Strings(have)
	if reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetRequestHeader(t *testing.T) {
	tests := []struct {
		name          string
		reqHeaders    api.HeaderMap
		headerName    string
		expectedOk    bool
		expectedValue string
	}{
		{
			name:          "single value",
			headerName:    "Content-Type",
			expectedOk:    true,
			expectedValue: "text/plain",
		},
		{
			name:          "comma value",
			headerName:    "Vary",
			expectedOk:    true,
			expectedValue: "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expectedOk: true,
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			reqHeaders := tc.reqHeaders
			if reqHeaders == nil {
				reqHeaders = newRequestHeader()
				addTestHeaders(reqHeaders)
			}
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: reqHeaders})

			value, ok := h.GetRequestHeader(ctx, tc.headerName)
			if want, have := tc.expectedValue, value; want != have {
				t.Errorf("unexpected header value, want: %v, have: %v", want, have)
			}
			if want, have := tc.expectedOk, ok; want != have {
				t.Errorf("unexpected header ok, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetRequestHeader(t *testing.T) {
	tests := []struct {
		name       string
		reqHeaders api.HeaderMap
		headerName string
		value      string
	}{
		{
			name:       "single value",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "single value overwrites",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "multi-field overwrites",
			headerName: "Set-Cookie",
			value:      "z",
		},
		{
			name:       "comma value",
			headerName: "X-Forwarded-For",
			value:      "1.2.3.4, 4.5.6.7",
		},
		{
			name:       "comma value overwrites",
			headerName: "Vary",
			value:      "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "aloha",
		},
		{
			name:       "empty value overwrites",
			headerName: "Empty",
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			reqHeaders := tc.reqHeaders
			if reqHeaders == nil {
				reqHeaders = newRequestHeader()
				addTestHeaders(reqHeaders)
			}
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{reqHeaders: reqHeaders})

			h.SetRequestHeader(ctx, tc.headerName, tc.value)
			if want, have := tc.value, strings.Join(headerValues(reqHeaders, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_GetResponseHeaderNames(t *testing.T) {
	respHeaders := newResponseHeader()
	addTestHeaders(respHeaders)
	ctx := variable.NewVariableContext(context.Background())
	ctx = context.WithValue(ctx, filterKey{}, &filter{respHeaders: respHeaders})

	want := []string{"Content-Type", "Vary", "Empty"}
	have := host{}.GetResponseHeaderNames(ctx)
	sort.Strings(have)
	if reflect.DeepEqual(want, have) {
		t.Errorf("unexpected header names, want: %v, have: %v", want, have)
	}
}

func Test_host_GetResponseHeader(t *testing.T) {
	tests := []struct {
		name          string
		respHeaders   api.HeaderMap
		headerName    string
		expectedOk    bool
		expectedValue string
	}{
		{
			name:          "single value",
			headerName:    "Content-Type",
			expectedOk:    true,
			expectedValue: "text/plain",
		},
		{
			name:          "comma value",
			headerName:    "Vary",
			expectedOk:    true,
			expectedValue: "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "Empty",
			expectedOk: true,
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			respHeaders := tc.respHeaders
			if respHeaders == nil {
				respHeaders = newResponseHeader()
				addTestHeaders(respHeaders)
			}
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{respHeaders: respHeaders})

			value, ok := h.GetResponseHeader(ctx, tc.headerName)
			if want, have := tc.expectedValue, value; want != have {
				t.Errorf("unexpected header value, want: %v, have: %v", want, have)
			}
			if want, have := tc.expectedOk, ok; want != have {
				t.Errorf("unexpected header ok, want: %v, have: %v", want, have)
			}
		})
	}
}

func Test_host_SetResponseHeader(t *testing.T) {
	tests := []struct {
		name        string
		respHeaders api.HeaderMap
		headerName  string
		value       string
	}{
		{
			name:       "single value",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "single value overwrites",
			headerName: "Accept",
			value:      "text/plain",
		},
		{
			name:       "comma value",
			headerName: "X-Forwarded-For",
			value:      "1.2.3.4, 4.5.6.7",
		},
		{
			name:       "comma value overwrites",
			headerName: "Vary",
			value:      "Accept-Encoding, User-Agent",
		},
		{
			name:       "empty value",
			headerName: "aloha",
		},
		{
			name:       "empty value overwrites",
			headerName: "Empty",
		},
		{
			name:       "no value",
			headerName: "Not Found",
		},
	}

	h := host{}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			respHeaders := tc.respHeaders
			if respHeaders == nil {
				respHeaders = newResponseHeader()
				addTestHeaders(respHeaders)
			}
			ctx := variable.NewVariableContext(context.Background())
			ctx = context.WithValue(ctx, filterKey{}, &filter{respHeaders: respHeaders})

			h.SetResponseHeader(ctx, tc.headerName, tc.value)
			if want, have := tc.value, strings.Join(headerValues(respHeaders, tc.headerName), "|"); want != have {
				t.Errorf("unexpected header, want: %v, have: %v", want, have)
			}
		})
	}
}

// Note: senders are supposed to concatenate multiple fields with the same
// name on comma, except the response header Set-Cookie. That said, a lot
// of middleware don't know about this and may repeat other headers anyway.
// See https://www.rfc-editoreqHeaders.org/rfc/rfc9110#section-5.2
func addTestHeaders(h api.HeaderMap) {
	h.Set("Content-Type", "text/plain")
	// TODO: multi-field doesn't work, yet
	// h.Add("Set-Cookie", "a")
	// h.Add("Set Cookie", "b")
	h.Set("Vary", "Accept-Encoding, User-Agent")
	h.Set("Empty", "")
}

func headerValues(h api.HeaderMap, name string) (values []string) {
	h.Range(func(key, value string) bool {
		if key == name {
			values = append(values, value)
		}
		return true
	})
	return values
}
