package filters

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type HttpMsgTransformerFilter struct {
	nextFilter Filter
}

func NewHttpMsgTransformerFilter() (*HttpMsgTransformerFilter, error) {
	return &HttpMsgTransformerFilter{}, nil
}

func (hmt *HttpMsgTransformerFilter) SetNextFilter(filter Filter) error {
	hmt.nextFilter = filter
	return nil
}

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // spelling per https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func (hmt *HttpMsgTransformerFilter) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	fmt.Println("HttpMsgTransformerFilter.Process() source Request: ----------------")
	rowSourceReq, err := httputil.DumpRequest(req, false)
	if err != nil {
		log.Fatal("HttpMsgTransformerFilter.Process() DumpRequest(): ", err)
	}
	fmt.Println(string(rowSourceReq))

	req, err = hmt.transformReqFromSourceToTarget(req)
	if err != nil {
		log.Fatal("Err: Faild to Transform Req From Source To Target: ", err)
		return err
	}

	fmt.Println("HttpMsgTransformerFilter.Process() target Request:")
	rowTrgReq, err := httputil.DumpRequest(req, false)
	if err != nil {
		log.Fatal("HttpMsgTransformerFilter.Process() DumpRequest(): ", err)
	}
	fmt.Println(string(rowTrgReq))

	hmt.nextFilter.Process(ctx, req, res)
	if err != nil {
		log.Fatal("Err: Faild to Call Process() on HttpMsgTransformer.nextFilter", err)
		return err
	}

	fmt.Println("HttpMsgTransformerFilter.Process() target Response:----------------------------")
	rowTrgRes, err := httputil.DumpResponse(res, false)
	if err != nil {
		log.Fatal("HttpMsgTransformerFilter.Process() DumpResponse(): ", err)
	}
	fmt.Println(string(rowTrgRes))

	res, err = hmt.transformResFromTargetToSource(res)
	if err != nil {
		log.Fatal("Err: Faild to Transform Res From Target To Source: ", err)
		return err
	}

	fmt.Println("HttpMsgTransformerFilter.Process() source Response: ")
	rowSourceRes, err := httputil.DumpResponse(res, false)
	if err != nil {
		log.Fatal("HttpMsgTransformerFilter.Process() DumpResponse(): ", err)
	}
	fmt.Println(string(rowSourceRes))

	return nil
}

func (hmt *HttpMsgTransformerFilter) transformReqFromSourceToTarget(sourceReq *http.Request) (trgtReq *http.Request, trsfrmErr error) {
	sourceReq.RequestURI = ""
	hmt.removeHopHeaders(sourceReq.Header)
	hmt.removeConnectionHeaders(sourceReq.Header)

	if clientIP, _, err := net.SplitHostPort(sourceReq.RemoteAddr); err == nil {
		hmt.appendHostToXForwardHeader(sourceReq.Header, clientIP)
	}
	trgtReq = sourceReq
	return trgtReq, nil
}

func (hmt *HttpMsgTransformerFilter) transformResFromTargetToSource(targetRes *http.Response) (sourceRes *http.Response, trnsfrmError error) {
	hmt.removeHopHeaders(targetRes.Header)
	hmt.removeConnectionHeaders(targetRes.Header)
	sourceRes = targetRes
	return sourceRes, nil
}

func (hmt *HttpMsgTransformerFilter) removeHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func (hmt *HttpMsgTransformerFilter) removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func (hmt *HttpMsgTransformerFilter) appendHostToXForwardHeader(header http.Header, host string) {
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}
