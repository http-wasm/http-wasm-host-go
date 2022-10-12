package mosn_test

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	requestBody  = "{\"hello\": \"panda\"}"
	responseBody = "{\"hello\": \"world\"}"

	serveJson = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
		w.Write([]byte(responseBody)) // nolint
	})

	servePath = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Content-Type", "text/plain")
		w.Write([]byte(r.URL.String())) // nolint
	})
)

type mosn struct {
	url    string
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	cmd    *exec.Cmd
}

func TestAuth(t *testing.T) {
	tests := []struct {
		hdr          http.Header
		status       int
		authenticate bool
	}{
		{
			hdr:          http.Header{"NotAuthorization": {"1"}},
			status:       http.StatusUnauthorized,
			authenticate: true,
		},
		{
			hdr:          http.Header{"Authorization": {""}},
			status:       http.StatusUnauthorized,
			authenticate: true,
		},
		{
			hdr:          http.Header{"Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}},
			status:       http.StatusOK,
			authenticate: false,
		},
		{
			hdr:          http.Header{"Authorization": {"0"}},
			status:       http.StatusUnauthorized,
			authenticate: false,
		},
	}

	backend := httptest.NewServer(serveJson)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), filepath.Join("examples", "auth.wasm"))
	defer mosn.cmd.Process.Kill()

	for _, tc := range tests {
		tt := tc
		t.Run(fmt.Sprintf("%s", tt.hdr), func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, mosn.url, nil)
			if err != nil {
				log.Panicln(err)
			}
			req.Header = tt.hdr

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Panicln(err)
			}
			resp.Body.Close()

			if got, want := resp.StatusCode, tt.status; got != want {
				t.Errorf("got %d, want %d", got, want)
			}
			if tt.authenticate {
				if auth, ok := resp.Header["Www-Authenticate"]; !ok {
					t.Error("Www-Authenticate header not found")
				} else if got, want := auth[0], "Basic realm=\"test\""; got != want {
					t.Errorf("got %s, want %s", got, want)
				}
			}
		})
	}
}

func TestLog(t *testing.T) {
	backend := httptest.NewServer(serveJson)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), filepath.Join("examples", "log.wasm"))
	defer mosn.cmd.Process.Kill()

	// Make a client request and print the contents to the same logger
	resp, err := http.Post(mosn.url, "application/json", strings.NewReader(requestBody))
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Ensure the response body was still readable!
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panicln(err)
	}
	if want, have := responseBody, string(body); want != have {
		log.Panicf("unexpected response body, want: %q, have: %q", want, have)
	}

	out := mosn.stdout.String()
	want := `
request body:
{"hello": "panda"}
response body:
{"hello": "world"}
`

	if !strings.Contains(out, strings.TrimSpace(want)) {
		t.Errorf("got %s, want %s", out, want)
	}
}

func TestProtocolVersion(t *testing.T) {
	tests := []struct {
		http2    bool
		respBody string
	}{
		{
			http2:    false,
			respBody: "HTTP/1.1",
		},
		// TODO(anuraaga): Enable http/2
	}

	backend := httptest.NewServer(serveJson)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), filepath.Join("tests", "protocol_version.wasm"))
	defer mosn.cmd.Process.Kill()

	for _, tc := range tests {
		tt := tc
		t.Run(tt.respBody, func(t *testing.T) {
			resp, err := http.Get(mosn.url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if want, have := tt.respBody, string(body); want != have {
				t.Errorf("unexpected response body, want: %q, have: %q", want, have)
			}
		})
	}
}

func TestRouter(t *testing.T) {
	tests := []struct {
		path     string
		respBody string
	}{
		{
			path:     "/",
			respBody: "hello world",
		},
		{
			path:     "/nothosst",
			respBody: "hello world",
		},
		{
			path:     "/host/a",
			respBody: "/a",
		},
		{
			path:     "/host/b?name=panda",
			respBody: "/b?name=panda",
		},
	}

	backend := httptest.NewServer(servePath)
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), filepath.Join("examples", "router.wasm"))
	defer mosn.cmd.Process.Kill()

	for _, tc := range tests {
		tt := tc
		t.Run(tt.path, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", mosn.url, tt.path)
			resp, err := http.Get(url)
			if err != nil {
				log.Panicln(err)
			}
			defer resp.Body.Close()
			content, _ := io.ReadAll(resp.Body)
			if got, want := string(content), tt.respBody; got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		body     string
		respBody string
	}{
		{
			body:     "open sesame",
			respBody: "###########",
		},
		{
			body:     "hello world",
			respBody: "hello world",
		},
		{
			body:     "hello open sesame world",
			respBody: "hello ########### world",
		},
	}

	var reqBody string
	var body string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := io.ReadAll(r.Body)
		reqBody = string(content)
		r.Header.Set("Content-Type", "text/plain")
		w.Write([]byte(body)) // nolint
	}))
	defer backend.Close()
	mosn := startMosn(t, backend.Listener.Addr().String(), filepath.Join("examples", "redact.wasm"))
	defer mosn.cmd.Process.Kill()

	for _, tc := range tests {
		tt := tc
		t.Run(tt.body, func(t *testing.T) {
			// body is both the request to the proxy and the response from the backend
			body = tt.body
			resp, err := http.Post(mosn.url, "text/plain", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			content, _ := io.ReadAll(resp.Body)
			if got, want := string(content), tt.respBody; got != want {
				t.Errorf("got %s, want %s", got, want)
			}
			if got, want := reqBody, tt.respBody; got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

func startMosn(t *testing.T, backendAddr string, wasm string) mosn {
	t.Helper()

	port := freePort()
	adminPort := freePort()

	configPath := filepath.Join(t.TempDir(), "config.json")
	f, err := os.Create(configPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	configTmpl, err := template.ParseFiles(filepath.Join("testdata", "config-tmpl.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = configTmpl.Execute(f, struct {
		Backend   string
		Wasm      string
		Port      int
		AdminPort int
	}{
		Backend:   backendAddr,
		Wasm:      filepath.Join("..", "..", "internal", "test", "testdata", wasm),
		Port:      port,
		AdminPort: adminPort,
	})
	if err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("./mosntest", "start", "--config", configPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", adminPort))
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return mosn{
				url:    fmt.Sprintf("http://localhost:%d", port),
				stdout: stdout,
				stderr: stderr,
				cmd:    cmd,
			}
		}
	}
	t.Fatal("mosn start failed")
	return mosn{}
}

func freePort() int {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func init() {
	goCmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goCmd, "build", "-o", "mosntest", "./internal/cmd")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if err := cmd.Run(); err != nil {
		log.Panicln(err)
	}
}
