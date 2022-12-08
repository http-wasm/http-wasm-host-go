package tck

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// GuestWASM is the guest wasm module used by the TCK. The host must load this
// module in the server being tested by Run.
//
//go:embed tck.wasm
var GuestWASM []byte

// Run executes the TCK. The url must point to a server with the TCK's guest
// wasm module loaded on top of the TCK's backend handler.
func Run(t *testing.T, url string) {
	if url == "" {
		t.Fatal("url is empty")
	}

	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}

	r := &testRunner{
		t:   t,
		url: url,
	}

	r.testGetMethod()
	r.testSetMethod()
	r.testGetURI()
	r.testSetURI()
	r.testGetRequestHeader()
	r.testGetRequestHeaderNames()
	r.testSetRequestHeader()
	r.testAddRequestHeader()
	r.testRemoveRequestHeader()
	r.testReadBody()
}

type testRunner struct {
	t   *testing.T
	url string
}

func (r *testRunner) testGetMethod() {
	// TODO(anuraaga): CONNECT is often handled outside of middleware
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS", "TRACE", "PATCH"}
	for _, method := range methods {
		testID := fmt.Sprintf("get-method/%s", method)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest(method, r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)
		})
	}
}

func (r *testRunner) testSetMethod() {
	testID := "set-method"
	r.t.Run(testID, func(t *testing.T) {
		req, err := http.NewRequest("GET", r.url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("x-httpwasm-tck-testid", testID)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		checkResponse(t, resp)

		if have, want := resp.Header.Get("x-httpwasm-next-method"), "POST"; have != want {
			t.Errorf("expected method to be %s, have %s", want, have)
		}
	})
}

func (r *testRunner) testGetURI() {
	tests := []struct {
		testID string
		uri    string
	}{
		{
			testID: "simple",
			uri:    "/simple",
		},
		{
			testID: "simple/escaping",
			uri:    "/simple%26clean",
		},
		{
			testID: "query",
			uri:    "/animal?name=panda",
		},
		{
			testID: "query/escaping",
			uri:    "/disney?name=chip%26dale",
		},
	}

	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("get-uri/%s", tt.testID)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", r.url, tt.uri), nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)
		})
	}
}

func (r *testRunner) testSetURI() {
	tests := []struct {
		testID string
		uri    string
	}{
		{
			testID: "simple",
			uri:    "/simple",
		},
		{
			testID: "simple/escaping",
			uri:    "/simple%26clean",
		},
		{
			testID: "query",
			uri:    "/animal?name=panda",
		},
		{
			testID: "query/escaping",
			uri:    "/disney?name=chip%26dale",
		},
	}

	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("set-uri/%s", tt.testID)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)

			if have, want := resp.Header.Get("x-httpwasm-next-uri"), tt.uri; have != want {
				t.Errorf("expected uri to be %s, have %s", want, have)
			}
		})
	}
}

func (r *testRunner) testGetRequestHeader() {
	tests := []struct {
		testID string
		key    string
		value  []string
	}{
		{
			testID: "lowercase-key",
			key:    "single-header",
			value:  []string{"value"},
		},
		{
			testID: "mixedcase-key",
			key:    "Single-Header",
			value:  []string{"value"},
		},
		{
			testID: "not-exists",
			key:    "not-header",
			value:  []string{},
		},
		{
			testID: "multiple-values",
			key:    "multi-header",
			value:  []string{"value1", "value2"},
		},
	}

	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("get-request-header/%s", tt.testID)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, v := range tt.value {
				req.Header.Add(tt.key, v)
			}

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)
		})
	}
}

func (r *testRunner) testGetRequestHeaderNames() {
	testID := "get-request-header-names"
	r.t.Run(testID, func(t *testing.T) {
		req, err := http.NewRequest("GET", r.url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("a-header", "value1")
		req.Header.Add("b-header", "2")

		req.Header.Set("x-httpwasm-tck-testid", testID)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}

		checkResponse(t, resp)
	})
}

func (r *testRunner) testReadBody() {
	tests := []struct {
		testID   string
		bodySize int
	}{
		{
			testID:   "empty",
			bodySize: 0,
		},
		{
			testID:   "small",
			bodySize: 5,
		},
		{
			testID:   "medium",
			bodySize: 2048,
		},
		{
			testID:   "large",
			bodySize: 4096,
		},
		{
			testID:   "xlarge",
			bodySize: 5000,
		},
	}

	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("read-body/%s", tt.testID)
		r.t.Run(testID, func(t *testing.T) {
			payload := strings.Repeat("a", tt.bodySize)
			req, err := http.NewRequest("POST", r.url, strings.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)
		})
	}
}

func (r *testRunner) testSetRequestHeader() {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "new",
			header: "new-header",
		},
		{
			name:   "existing",
			header: "existing-header",
		},
	}
	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("set-request-header/%s", tt.name)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("existing-header", "bear")

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)

			if have, want := resp.Header.Get(fmt.Sprintf("x-httpwasm-next-header-%s-0", tt.header)), "value"; have != want {
				t.Errorf("expected header to be %s, have %s", want, have)
			}
		})
	}
}

func (r *testRunner) testAddRequestHeader() {
	tests := []struct {
		name   string
		header string
		values []string
	}{
		{
			name:   "new",
			header: "new-header",
			values: []string{"value"},
		},
		{
			name:   "existing",
			header: "existing-header",
			values: []string{"bear", "value"},
		},
	}
	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("add-request-header/%s", tt.name)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("existing-header", "bear")

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)

			for i, v := range tt.values {
				if have, want := resp.Header.Get(fmt.Sprintf("x-httpwasm-next-header-%s-%d", tt.header, i)), v; have != want {
					t.Errorf("expected header to be %s, have %s", want, have)
				}
			}
		})
	}
}

func (r *testRunner) testRemoveRequestHeader() {
	tests := []struct {
		name    string
		header  string
		removed bool
	}{
		{
			name:    "new",
			header:  "new-header",
			removed: false,
		},
		{
			name:    "existing",
			header:  "existing-header",
			removed: true,
		},
	}
	for _, tc := range tests {
		tt := tc
		testID := fmt.Sprintf("remove-request-header/%s", tt.name)
		r.t.Run(testID, func(t *testing.T) {
			req, err := http.NewRequest("GET", r.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("existing-header", "bear")

			req.Header.Set("x-httpwasm-tck-testid", testID)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
			}

			checkResponse(t, resp)

			want := "bear"
			if tt.removed {
				want = ""
			}
			if have := resp.Header.Get("x-httpwasm-next-header-existing-header-0"); have != want {
				t.Errorf("expected header to be %s, have %s", want, have)
			}
		})
	}
}

func checkResponse(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp.Header.Get("x-httpwasm-tck-handled") != "1" {
		t.Error("x-httpwasm-tck-handled header is missing")
	}

	if resp.StatusCode == http.StatusInternalServerError {
		msg := resp.Header.Get("x-httpwasm-tck-failed")
		if msg == "" {
			t.Error("error status without test failure message")
		}
		t.Errorf("assertion failed: %s", msg)
	}
}
