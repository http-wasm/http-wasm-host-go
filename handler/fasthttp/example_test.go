package wasm

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/valyala/fasthttp"

	httpwasm "github.com/http-wasm/http-wasm-host-go"
	"github.com/http-wasm/http-wasm-host-go/internal/test"
)

var serveJson = func(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.WriteString("{\"hello\": \"world\"}") // nolint
	ctx.SetStatusCode(200)
}

var servePath = func(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "text/plain")
	ctx.Write(ctx.Path()) // nolint
	ctx.SetStatusCode(200)
}

func Example_auth() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an auth interceptor.
	mw, err := NewMiddleware(ctx, test.AuthWasm)
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Wrap the real handler with an interceptor implemented in WebAssembly.
	wrapped, err := mw.NewHandler(ctx, serveJson)
	if err != nil {
		log.Panicln(err)
	}
	defer wrapped.Close(ctx)

	// Start the server with the wrapped handler.
	ts, url := listenAndServe(wrapped)
	defer ts.Shutdown()

	// Invoke some requests, only one of which should pass
	headers := []http.Header{
		{"NotAuthorization": {"1"}},
		{"Authorization": {""}},
		{"Authorization": {"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}},
		{"Authorization": {"0"}},
	}

	for _, header := range headers {
		req, err := http.NewRequest(http.MethodGet, url, nil)
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
	}

	// Output:
	// Unauthorized
	// Unauthorized
	// OK
	// Unauthorized
}

func Example_log() {
	ctx := context.Background()
	logger := func(_ context.Context, message string) { fmt.Println(message) }

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// a logging interceptor.
	mw, err := NewMiddleware(ctx, test.LogWasm, httpwasm.Logger(logger))
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Wrap the real handler with an interceptor implemented in WebAssembly.
	wrapped, err := mw.NewHandler(ctx, serveJson)
	if err != nil {
		log.Panicln(err)
	}
	defer wrapped.Close(ctx)

	// Start the server with the wrapped handler.
	ts, url := listenAndServe(wrapped)
	defer ts.Shutdown()

	// Make a client request and print the contents to the same logger
	resp, err := http.Get(url)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()
	content, _ := io.ReadAll(resp.Body)
	logger(ctx, string(content))

	// Output:
	// before
	// after
	// {"hello": "world"}
}

func Example_router() {
	ctx := context.Background()

	// Configure and compile the WebAssembly guest binary. In this case, it is
	// an example request router.
	mw, err := NewMiddleware(ctx, test.RouterWasm)
	if err != nil {
		log.Panicln(err)
	}
	defer mw.Close(ctx)

	// Wrap the real handler with an interceptor implemented in WebAssembly.
	wrapped, err := mw.NewHandler(ctx, servePath)
	if err != nil {
		log.Panicln(err)
	}
	defer wrapped.Close(ctx)

	// Start the server with the wrapped handler.
	ts, url := listenAndServe(wrapped)
	defer ts.Shutdown()

	// Invoke some requests, only one of which should pass
	paths := []string{
		"/",
		"/nothosst",
		"/host/a",
	}

	for _, p := range paths {
		resp, err := http.Get(fmt.Sprintf("%s/%s", url, p))
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

func listenAndServe(wrapped RequestHandler) (*fasthttp.Server, string) {
	ts := &fasthttp.Server{Handler: wrapped.Handle}
	ln, err := net.Listen("tcp4", "127.0.0.1:")
	if err != nil {
		log.Panicln(err)
	}
	url := fmt.Sprintf("http://%s", ln.Addr().String())
	go func() {
		if err = ts.Serve(ln); err != nil {
			log.Panicln(err)
		}
	}()
	return ts, url
}
