package tck

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
)

// BackendHandler is a http.Handler implementing the logic expected by the TCK.
// It serves to echo back information from the request to the response for
// checking expectations.
func BackendHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-httpwasm-next-method", r.Method)
		w.Header().Set("x-httpwasm-next-uri", r.RequestURI)
		for k, vs := range r.Header {
			for i, v := range vs {
				w.Header().Add(fmt.Sprintf("x-httpwasm-next-header-%s-%d", k, i), v)
			}
		}
	})
}

// StartBackend starts a httptest.Server at the given address implementing BackendHandler.
func StartBackend(addr string) *httptest.Server {
	s := httptest.NewUnstartedServer(BackendHandler())
	if addr != "" {
		s.Listener.Close()
		l, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		s.Listener = l
	}
	s.Start()
	return s
}
