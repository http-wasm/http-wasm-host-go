package main

import (
	"bytes"
	"fmt"
	"strings"

	httpwasm "github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

// TODO(anuraaga): enable_features, get_header, set_header need to be tested separately.

func main() {
	enabledFeatures := httpwasm.Host.EnableFeatures(api.FeatureBufferRequest | api.FeatureBufferResponse | api.FeatureTrailers)
	h := handler{enabledFeatures: enabledFeatures}

	httpwasm.HandleRequestFn = h.handleRequest
}

type handler struct {
	enabledFeatures api.Features
}

func (h *handler) handleRequest(req api.Request, resp api.Response) (next bool, reqCtx uint32) {
	testID, _ := req.Headers().Get("x-httpwasm-tck-testid")
	if len(testID) == 0 {
		resp.SetStatusCode(500)
		resp.Body().WriteString("missing x-httpwasm-tck-testid header")
		return
	}

	switch testID {
	case "get-method/GET":
		next, reqCtx = h.testGetMethod(req, resp, "GET")
	case "get-method/HEAD":
		next, reqCtx = h.testGetMethod(req, resp, "HEAD")
	case "get-method/POST":
		next, reqCtx = h.testGetMethod(req, resp, "POST")
	case "get-method/PUT":
		next, reqCtx = h.testGetMethod(req, resp, "PUT")
	case "get-method/DELETE":
		next, reqCtx = h.testGetMethod(req, resp, "DELETE")
	case "get-method/CONNECT":
		next, reqCtx = h.testGetMethod(req, resp, "CONNECT")
	case "get-method/OPTIONS":
		next, reqCtx = h.testGetMethod(req, resp, "OPTIONS")
	case "get-method/TRACE":
		next, reqCtx = h.testGetMethod(req, resp, "TRACE")
	case "get-method/PATCH":
		next, reqCtx = h.testGetMethod(req, resp, "PATCH")
	case "set-method":
		next, reqCtx = h.testSetMethod(req, resp)
	case "get-uri/simple":
		next, reqCtx = h.testGetURI(req, resp, "/simple")
	case "get-uri/simple/escaping":
		next, reqCtx = h.testGetURI(req, resp, "/simple%26clean")
	case "get-uri/query":
		next, reqCtx = h.testGetURI(req, resp, "/animal?name=panda")
	case "get-uri/query/escaping":
		next, reqCtx = h.testGetURI(req, resp, "/disney?name=chip%26dale")
	case "set-uri/simple":
		next, reqCtx = h.testSetURI(req, resp, "/simple")
	case "set-uri/simple/escaping":
		next, reqCtx = h.testSetURI(req, resp, "/simple%26clean")
	case "set-uri/query":
		next, reqCtx = h.testSetURI(req, resp, "/animal?name=panda")
	case "set-uri/query/escaping":
		next, reqCtx = h.testSetURI(req, resp, "/disney?name=chip%26dale")
	case "get-request-header/lowercase-key":
		next, reqCtx = h.testGetRequestHeader(req, resp, "single-header", []string{"value"})
	case "get-request-header/mixedcase-key":
		next, reqCtx = h.testGetRequestHeader(req, resp, "Single-Header", []string{"value"})
	case "get-request-header/not-exists":
		next, reqCtx = h.testGetRequestHeader(req, resp, "not-header", []string{})
	case "get-request-header/multiple-values":
		next, reqCtx = h.testGetRequestHeader(req, resp, "multi-header", []string{"value1", "value2"})
	case "get-request-header-names":
		next, reqCtx = h.testGetRequestHeaderNames(req, resp, []string{"a-header", "b-header"})
	case "set-request-header/new":
		next, reqCtx = h.testSetRequestHeader(req, resp, "new-header", "value")
	case "set-request-header/existing":
		next, reqCtx = h.testSetRequestHeader(req, resp, "existing-header", "value")
	case "add-request-header/new":
		next, reqCtx = h.testAddRequestHeader(req, resp, "new-header", "value")
	case "add-request-header/existing":
		next, reqCtx = h.testAddRequestHeader(req, resp, "existing-header", "value")
	case "remove-request-header/new":
		next, reqCtx = h.testRemoveRequestHeader(req, resp, "new-header")
	case "remove-request-header/existing":
		next, reqCtx = h.testRemoveRequestHeader(req, resp, "existing-header")
	case "read-body/empty":
		next, reqCtx = h.testReadBody(req, resp, "")
	case "read-body/small":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 5))
	case "read-body/medium":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 2048))
	case "read-body/large":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 4096))
	case "read-body/xlarge":
		next, reqCtx = h.testReadBody(req, resp, strings.Repeat("a", 5000))
	default:
		fail(resp, "unknown x-httpwasm-test-id")
	}

	resp.Headers().Set("x-httpwasm-tck-handled", "1")

	return
}

func (h *handler) testGetMethod(req api.Request, resp api.Response, expectedMethod string) (next bool, reqCtx uint32) {
	if req.GetMethod() != expectedMethod {
		fail(resp, fmt.Sprintf("get_method: expected %s, have %s", expectedMethod, req.GetMethod()))
	}
	return
}

func (h *handler) testSetMethod(req api.Request, _ api.Response) (next bool, reqCtx uint32) {
	req.SetMethod("POST")
	return true, 0
}

func (h *handler) testGetURI(req api.Request, resp api.Response, expectedURI string) (next bool, reqCtx uint32) {
	if req.GetURI() != expectedURI {
		fail(resp, fmt.Sprintf("get_uri: expected %s, have %s", expectedURI, req.GetURI()))
	}
	return
}

func (h *handler) testSetURI(req api.Request, resp api.Response, uri string) (next bool, reqCtx uint32) {
	req.SetURI(uri)
	return true, 0
}

func (h *handler) testGetRequestHeader(req api.Request, resp api.Response, header string, expectedValue []string) (next bool, reqCtx uint32) {
	have := req.Headers().GetAll(header)
	if len(have) != len(expectedValue) {
		fail(resp, fmt.Sprintf("get_request_header: expected %d values, have %d", len(expectedValue), len(have)))
		return
	}
	for i, v := range have {
		if v != expectedValue[i] {
			fail(resp, fmt.Sprintf("get_request_header: expected %s, have %s", expectedValue[i], v))
			return
		}
	}

	return
}

func (h *handler) testGetRequestHeaderNames(req api.Request, resp api.Response, expectedNames []string) (next bool, reqCtx uint32) {
	have := req.Headers().Names()

	// Don't check an exact match since it can be tricky to control automatic headers like user-agent, we're probably
	// fine as long as we have all the expected headers.
	// TODO(anuraaga): Confirm this suspicion

	for _, name := range expectedNames {
		found := false
		for _, haveName := range have {
			if name == haveName {
				found = true
				break
			}
		}
		if !found {
			fail(resp, fmt.Sprintf("get_request_header_names: expected %s, not found. have: %v", name, have))
			return
		}
	}

	return
}

func (h *handler) testSetRequestHeader(req api.Request, resp api.Response, header string, value string) (next bool, reqCtx uint32) {
	req.Headers().Set(header, value)
	return true, 0
}

func (h *handler) testAddRequestHeader(req api.Request, resp api.Response, header string, value string) (next bool, reqCtx uint32) {
	req.Headers().Add(header, value)
	return true, 0
}

func (h *handler) testRemoveRequestHeader(req api.Request, resp api.Response, header string) (next bool, reqCtx uint32) {
	req.Headers().Remove(header)
	return true, 0
}

func (h *handler) testReadBody(req api.Request, resp api.Response, expectedBody string) (next bool, reqCtx uint32) {
	body := req.Body()
	buf := &bytes.Buffer{}
	sz, err := body.WriteTo(buf)
	if err != nil {
		fail(resp, fmt.Sprintf("read_body: error %v", err))
		return
	}

	if int(sz) != len(expectedBody) {
		fail(resp, fmt.Sprintf("read_body: expected %d bytes, have %d", len(expectedBody), sz))
		return
	}

	if buf.String() != expectedBody {
		fail(resp, fmt.Sprintf("read_body: expected %s, have %s", expectedBody, buf.String()))
		return
	}

	return
}

func fail(resp api.Response, msg string) {
	resp.SetStatusCode(500)
	resp.Headers().Set("x-httpwasm-tck-failed", msg)
}
