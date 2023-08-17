package main

import (
	"bytes"
	"fmt"
	"strings"

	httpwasm "github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

// TODO: enable_features, get_header, set_header need to be tested separately.

func main() {
	enabledFeatures := httpwasm.Host.EnableFeatures(api.FeatureBufferRequest | api.FeatureBufferResponse | api.FeatureTrailers)
	h := handler{enabledFeatures: enabledFeatures}

	httpwasm.HandleRequestFn = h.handleRequest
	httpwasm.HandleResponseFn = h.handleResponse
}

type handler struct {
	enabledFeatures api.Features
	protocolVersion string
}

func (h *handler) handleRequest(req api.Request, resp api.Response) (next bool, reqCtx uint32) {
	testID, _ := req.Headers().Get("x-httpwasm-tck-testid")
	if len(testID) == 0 {
		resp.SetStatusCode(500)
		resp.Body().WriteString("missing x-httpwasm-tck-testid header")
		return
	}

	switch testID {
	case "get_protocol_version":
		// The test runner compares this with the request value.
		h.protocolVersion = req.GetProtocolVersion()
		return true, 0
	case "get_method/GET":
		next, reqCtx = h.testGetMethod(req, resp, "GET")
	case "get_method/HEAD":
		next, reqCtx = h.testGetMethod(req, resp, "HEAD")
	case "get_method/POST":
		next, reqCtx = h.testGetMethod(req, resp, "POST")
	case "get_method/PUT":
		next, reqCtx = h.testGetMethod(req, resp, "PUT")
	case "get_method/DELETE":
		next, reqCtx = h.testGetMethod(req, resp, "DELETE")
	case "get_method/CONNECT":
		next, reqCtx = h.testGetMethod(req, resp, "CONNECT")
	case "get_method/OPTIONS":
		next, reqCtx = h.testGetMethod(req, resp, "OPTIONS")
	case "get_method/TRACE":
		next, reqCtx = h.testGetMethod(req, resp, "TRACE")
	case "get_method/PATCH":
		next, reqCtx = h.testGetMethod(req, resp, "PATCH")
	case "set_method":
		next, reqCtx = h.testSetMethod(req, resp)
	case "get_uri/simple":
		next, reqCtx = h.testGetURI(req, resp, "/simple")
	case "get_uri/simple/escaping":
		next, reqCtx = h.testGetURI(req, resp, "/simple%26clean")
	case "get_uri/query":
		next, reqCtx = h.testGetURI(req, resp, "/animal?name=panda")
	case "get_uri/query/escaping":
		next, reqCtx = h.testGetURI(req, resp, "/disney?name=chip%26dale")
	case "set_uri/simple":
		next, reqCtx = h.testSetURI(req, resp, "/simple")
	case "set_uri/simple/escaping":
		next, reqCtx = h.testSetURI(req, resp, "/simple%26clean")
	case "set_uri/query":
		next, reqCtx = h.testSetURI(req, resp, "/animal?name=panda")
	case "set_uri/query/escaping":
		next, reqCtx = h.testSetURI(req, resp, "/disney?name=chip%26dale")
	case "get_header_values/request/lowercase-key":
		next, reqCtx = h.testGetRequestHeader(req, resp, "single-header", []string{"value"})
	case "get_header_values/request/mixedcase-key":
		next, reqCtx = h.testGetRequestHeader(req, resp, "Single-Header", []string{"value"})
	case "get_header_values/request/not-exists":
		next, reqCtx = h.testGetRequestHeader(req, resp, "not-header", []string{})
	case "get_header_values/request/multiple-values":
		next, reqCtx = h.testGetRequestHeader(req, resp, "multi-header", []string{"value1", "value2"})
	case "get_header_names/request":
		next, reqCtx = h.testGetRequestHeaderNames(req, resp, []string{"a-header", "b-header"})
	case "set_header_value/request/new":
		next, reqCtx = h.testSetRequestHeader(req, resp, "new-header", "value")
	case "set_header_value/request/existing":
		next, reqCtx = h.testSetRequestHeader(req, resp, "existing-header", "value")
	case "add_header_value/request/new":
		next, reqCtx = h.testAddRequestHeader(req, resp, "new-header", "value")
	case "add_header_value/request/existing":
		next, reqCtx = h.testAddRequestHeader(req, resp, "existing-header", "value")
	case "remove_header/request/new":
		next, reqCtx = h.testRemoveRequestHeader(req, resp, "new-header")
	case "remove_header/request/existing":
		next, reqCtx = h.testRemoveRequestHeader(req, resp, "existing-header")
	case "read_body/request/empty":
		next, reqCtx = h.testReadBody(req, resp, "")
	case "read_body/request/small":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 5))
	case "read_body/request/medium":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 2048))
	case "read_body/request/large":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 4096))
	case "read_body/request/xlarge":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 5000))
	default:
		fail(resp, "unknown x-httpwasm-test-id")
	}

	resp.Headers().Set("x-httpwasm-tck-handled", "1")

	return
}

func (h *handler) testGetMethod(req api.Request, resp api.Response, expectedMethod string) (next bool, reqCtx uint32) {
	if req.GetMethod() != expectedMethod {
		fail(resp, fmt.Sprintf("get_method: want %s, have %s", expectedMethod, req.GetMethod()))
	}
	return true, 0
}

func (h *handler) testSetMethod(req api.Request, _ api.Response) (next bool, reqCtx uint32) {
	req.SetMethod("POST")
	return true, 0
}

func (h *handler) testGetURI(req api.Request, resp api.Response, expectedURI string) (next bool, reqCtx uint32) {
	if req.GetURI() != expectedURI {
		fail(resp, fmt.Sprintf("get_uri: want %s, have %s", expectedURI, req.GetURI()))
	}
	return true, 0
}

func (h *handler) testSetURI(req api.Request, _ api.Response, uri string) (next bool, reqCtx uint32) {
	req.SetURI(uri)
	return true, 0
}

func (h *handler) testGetRequestHeader(req api.Request, resp api.Response, header string, expectedValue []string) (next bool, reqCtx uint32) {
	have := req.Headers().GetAll(header)
	if len(have) != len(expectedValue) {
		fail(resp, fmt.Sprintf("get_request_header: want %d values, have %d", len(expectedValue), len(have)))
		return
	}
	for i, v := range have {
		if v != expectedValue[i] {
			fail(resp, fmt.Sprintf("get_request_header: want %s, have %s", expectedValue[i], v))
			return
		}
	}

	return true, 0
}

func (h *handler) testGetRequestHeaderNames(req api.Request, resp api.Response, expectedNames []string) (next bool, reqCtx uint32) {
	have := req.Headers().Names()

	// Don't check an exact match since it can be tricky to control automatic headers like user-agent, we're probably
	// fine as long as we have all the want headers.
	// TODO: Confirm this suspicion

	for _, name := range expectedNames {
		found := false
		for _, haveName := range have {
			if name == haveName {
				found = true
				break
			}
		}
		if !found {
			fail(resp, fmt.Sprintf("get_header_names/request: want %s, not found. have: %v", name, have))
			return
		}
	}

	return true, 0
}

func (h *handler) testSetRequestHeader(req api.Request, _ api.Response, header string, value string) (next bool, reqCtx uint32) {
	req.Headers().Set(header, value)
	return true, 0
}

func (h *handler) testAddRequestHeader(req api.Request, _ api.Response, header string, value string) (next bool, reqCtx uint32) {
	req.Headers().Add(header, value)
	return true, 0
}

func (h *handler) testRemoveRequestHeader(req api.Request, _ api.Response, header string) (next bool, reqCtx uint32) {
	req.Headers().Remove(header)
	return true, 0
}

func (h *handler) testReadBody(req api.Request, resp api.Response, expectedBody string) (next bool, reqCtx uint32) {
	body := req.Body()
	buf := &bytes.Buffer{}
	sz, err := body.WriteTo(buf)
	if err != nil {
		fail(resp, fmt.Sprintf("read_body/request: error %v", err))
		return
	}

	if int(sz) != len(expectedBody) {
		fail(resp, fmt.Sprintf("read_body/request: want %d bytes, have %d", len(expectedBody), sz))
		return
	}

	if buf.String() != expectedBody {
		fail(resp, fmt.Sprintf("read_body/request: want %s, have %s", expectedBody, buf.String()))
		return
	}

	return true, 0
}

func fail(resp api.Response, msg string) {
	resp.SetStatusCode(500)
	resp.Headers().Set("x-httpwasm-tck-failed", msg)
}

func (h *handler) handleResponse(ctx uint32, _ api.Request, res api.Response, err bool) {
	res.Headers().Set("x-httpwasm-tck-handled", "1")
	res.Body().WriteString(h.protocolVersion)
}
