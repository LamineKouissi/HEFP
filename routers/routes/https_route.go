package routes

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

type HttpsRoute struct {
}

func NewHttspRoute() (*HttpsRoute, error) {
	return &HttpsRoute{}, nil
}

func (hs *HttpsRoute) HandleF(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	go hs.transfer(ctx, destConn, clientConn)
	go hs.transfer(ctx, clientConn, destConn)
}

func (hs *HttpsRoute) transfer(cxt context.Context, destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
