package wasm_test

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
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

		resp, err := http.DefaultClient.Do(req)
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
			fmt.Println("Www-Authenticate: ", auth[0])
		}
	}

	// Output:
	// Unauthorized
	// Www-Authenticate:  Basic realm="test"
	// Unauthorized
	// Www-Authenticate:  Basic realm="test"
	// OK
	// Unauthorized
}

func Example_log() {
	ctx := context.Background()
	logger := func(_ context.Context, message string) { fmt.Println(message) }

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// a logging interceptor.
	mw, err := wasm.NewMiddleware(ctx, test.BinExampleLog, httpwasm.Logger(logger))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Create the real request handler.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ensure the request body is readable
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Panicln(err)
		}
		if want, have := requestBody, string(body); want != have {
			log.Panicf("unexpected request body, want: %q, have: %q", want, have)
		}
		r.Header.Set("Content-Type", "application/json")
		w.Write([]byte(responseBody)) // nolint
	})

	// Wrap this with an interceptor implemented in WebAssembly.
	wrapped := mw.NewHandler(ctx, next)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	// Make a client request and print the contents to the same logger
	resp, err := ts.Client().Post(ts.URL, "application/json", strings.NewReader(requestBody))
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

	// Output:
	// request body:
	// {"hello": "panda"}
	// response body:
	// {"hello": "world"}
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

	// The below proves redaction worked for both request and response bodies!

	// Output:
	// ###########
	// ###########
	// hello world
	// hello world
	// hello ########### world
	// hello ########### world
}
