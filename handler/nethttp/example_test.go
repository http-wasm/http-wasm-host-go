package wasm_test

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/tetratelabs/wazero"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/api"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
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
		w.Write([]byte(r.URL.Path)) // nolint
	})
)

func Example_auth() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an auth interceptor.
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleAuth)
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := serveJson

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, next)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Invoke some requests, only one of which should pass
	headers := []http.Header{
		{"NotAuthorization": {"1"}},
		{"Authorization": {""}},
		{"Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}},
		{"Authorization": {"0"}},
	}

	for _, header := range headers {
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		if err != nil {
			log.Panicln(err)
		}
		req.Header = header

		resp, err := ts.Client().Do(req)
		if err != nil {
			log.Panicln(err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			fmt.Println("OK")
		case http.StatusUnauthorized:
			fmt.Println("Unauthorized")
		default:
			log.Panicln("unexpected status code", resp.StatusCode)
		}
		if auth, ok := resp.Header["Www-Authenticate"]; ok {
			fmt.Println("Www-Authenticate:", auth[0])
		}
	}

	// Output:
	// Unauthorized
	// Www-Authenticate: Basic realm="test"
	// Unauthorized
	// OK
	// Unauthorized
}

func Example_wasi() {
	ctx := context.Background()
	moduleConfig := wazero.NewModuleConfig().WithStdout(os.Stdout)

	// Configure and compile the WebAssembly guest binary. In this case, it
	// prints the request and response to the STDOUT via WASI.
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleWASI,
		httpwasm.ModuleConfig(moduleConfig))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=b") // example of multiple headers
		w.Header().Add("Set-Cookie", "c=d")

		// Use chunked encoding so we can set a test trailer
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Trailer", "grpc-status")
		w.Header().Set(http.TrailerPrefix+"grpc-status", "1")
		w.Write([]byte(`{"hello": "world"}`)) // nolint
	})

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, next)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Make a client request which should print to the console
	req, err := http.NewRequest("POST", ts.URL, strings.NewReader(requestBody))
	if err != nil {
		log.Panicln(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Host = "localhost"
	resp, err := ts.Client().Do(req)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Output:
	// POST / HTTP/1.1
	// Accept-Encoding: gzip
	// Content-Length: 18
	// Content-Type: application/json
	// Host: localhost
	// User-Agent: Go-http-client/1.1
	//
	// {"hello": "panda"}
	//
	// HTTP/1.1 200
	// Content-Type: application/json
	// Set-Cookie: a=b
	// Set-Cookie: c=d
	// Trailer: grpc-status
	// Transfer-Encoding: chunked
	//
	// {"hello": "world"}
	// grpc-status: 1
}

func Example_log() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// a logging interceptor.
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleLog, httpwasm.Logger(api.ConsoleLogger{}))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := serveJson

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, next)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Make a client request.
	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	// Output:
	// hello world
}

func Example_router() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an example request router.
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleRouter)
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Wrap the real handler with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, servePath)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Invoke some requests, only one of which should pass
	paths := []string{
		"",
		"nothosst",
		"host/a",
	}

	for _, p := range paths {
		url := fmt.Sprintf("%s/%s", ts.URL, p)
		resp, err := ts.Client().Get(url)
		if err != nil {
			log.Panicln(err)
		}
		defer resp.Body.Close()
		content, _ := io.ReadAll(resp.Body)
		fmt.Println(string(content))
	}

	// Output:
	// hello world
	// hello world
	// /a
}

func Example_redact() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an example response redact.
	secret := "open sesame"
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleRedact,
		httpwasm.GuestConfig([]byte(secret)))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	var body string
	serveBody := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := io.ReadAll(r.Body)
		fmt.Println(string(content))
		r.Header.Set("Content-Type", "text/plain")
		w.Write([]byte(body)) // nolint
	})

	// Wrap the real handler with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, serveBody)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	bodies := []string{
		secret,
		"hello world",
		fmt.Sprintf("hello %s world", secret),
	}

	for _, b := range bodies {
		body = b

		resp, err := ts.Client().Post(ts.URL, "text/plain", strings.NewReader(body))
		if err != nil {
			log.Panicln(err)
		}
		defer resp.Body.Close()
		content, _ := io.ReadAll(resp.Body)
		fmt.Println(string(content))
	}

	// Output:
	// ###########
	// ###########
	// hello world
	// hello world
	// hello ########### world
	// hello ########### world
}
