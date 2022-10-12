package mosn_test

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/test/util/mosn"
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

type testMosn struct {
	url     string
	logPath string
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
	cmd     *exec.Cmd
	*mosn.MosnWrapper
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
	defer mosn.Close()

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
	want := []string{"wasm: request body:", `wasm: {"hello": "panda"}`, "wasm: response body:", `wasm: {"hello": "world"}`}

	var missing []string
	for _, w := range want {
		if !strings.Contains(out, w) {
			missing = append(missing, w)
		}
	}

	if len(missing) > 0 {
		t.Errorf("got %s, missing: %s", out, missing)
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

func startMosn(t *testing.T, backendAddr string, wasm string) testMosn {
	t.Helper()

	port := freePort()
	adminPort := freePort()

	logPath := filepath.Join(t.TempDir(), "mosn.log")

	app := mosn.NewMosn(&v2.MOSNConfig{
		Servers: []v2.ServerConfig{
			{
				DefaultLogPath: logPath,
				Routers: []*v2.RouterConfiguration{
					{
						RouterConfigurationConfig: v2.RouterConfigurationConfig{
							RouterConfigName: "server_router",
						},
						VirtualHosts: []v2.VirtualHost{
							{
								Name:    "serverHost",
								Domains: []string{"*"},
								Routers: []v2.Router{
									{
										RouterConfig: v2.RouterConfig{
											Match: v2.RouterMatch{
												Prefix: "/",
											},
											Route: v2.RouteAction{
												RouterActionConfig: v2.RouterActionConfig{
													ClusterName: "serverCluster",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Listeners: []v2.Listener{
					{
						ListenerConfig: v2.ListenerConfig{
							Name:       "serverListener",
							AddrConfig: fmt.Sprintf("127.0.0.1:%d", port),
							BindToPort: true,
							FilterChains: []v2.FilterChain{
								{
									FilterChainConfig: v2.FilterChainConfig{
										Filters: []v2.Filter{
											{
												Type: "proxy",
												Config: map[string]interface{}{
													"downstream_protocol": "Http1",
													"upstream_protocol":   "Http1",
													"router_config_name":  "server_router",
												},
											},
										},
									},
								},
							},
							StreamFilters: []v2.Filter{
								{
									Type: "httpwasm",
									Config: map[string]interface{}{
										"path":   filepath.Join("..", "..", "internal", "test", "testdata", wasm),
										"config": "open sesame",
									},
								},
							},
						},
					},
				},
			},
		},
		ClusterManager: v2.ClusterManagerConfig{
			Clusters: []v2.Cluster{
				{
					Name:                 "serverCluster",
					ClusterType:          "SIMPLE",
					LbType:               "LB_RANDOM",
					MaxRequestPerConn:    1024,
					ConnBufferLimitBytes: 32768,
					Hosts: []v2.Host{
						{
							HostConfig: v2.HostConfig{
								Address: backendAddr,
							},
						},
					},
				},
			},
		},
		RawAdmin: &v2.Admin{
			Address: &v2.AddressInfo{
				SocketAddress: v2.SocketAddress{
					Address:   "0.0.0.0",
					PortValue: uint32(adminPort),
				},
			},
		},
		DisableUpgrade: true,
	})

	app.Start()
	for i := 0; i < 100; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", adminPort))
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return testMosn{
				url:         fmt.Sprintf("http://localhost:%d", port),
				logPath:     logPath,
				MosnWrapper: app,
			}
		}
	}
	t.Fatal("mosn start failed")
	return testMosn{}
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
