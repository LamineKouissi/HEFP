package routes

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/LamineKouissi/LHP/filters"
)

type HttpRoute struct {
	HttpFilterChaine filters.Filter
}

func NewHttpRoute(filterChaine filters.Filter) (*HttpRoute, error) {
	if filterChaine == nil {
		return nil, errors.New("nil filterChaine")
	}
	return &HttpRoute{HttpFilterChaine: filterChaine}, nil
}

func (h *HttpRoute) SetHttpFilterChaine(hfc filters.Filter) error {
	if hfc == nil {
		return errors.New("nil filterChaine")
	}
	h.HttpFilterChaine = hfc
	return nil
}

func (h *HttpRoute) HandleF(ctx context.Context, w http.ResponseWriter, req *http.Request) {

	resp := &http.Response{}
	err := h.HttpFilterChaine.Process(ctx, req, resp)
	if err != nil {
		log.Println(err)
	}

	// resBytes, err := httputil.DumpResponse(resp, false)
	// if err != nil {
	// 	log.Fatal("DumpResponse")
	// }
	// log.Println("handleHTTP() Res: -------------")
	// fmt.Println(string(resBytes))

	defer resp.Body.Close()
	h.copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (hs *HttpRoute) copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
