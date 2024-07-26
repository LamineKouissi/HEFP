package connectors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

type HttpsConnector struct {
	client *http.Client
}

func (usc *HttpsConnector) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	fmt.Println("HttpsConnector.Process() target Request: ----------------")
	rowReq, err := httputil.DumpRequest(req, false)
	if err != nil {
		log.Fatal("HttpsConnector.Process() DumpRequest(): ", err)
	}
	fmt.Println(string(rowReq))

	trgtRes, err := usc.client.Do(req)
	if err != nil {
		log.Fatal("Err: Faild to Fire Req to Target through: HttpsConnector.Process() : ", err)
		return err
	}

	*res = *trgtRes

	log.Println("HttpsConnector.Process() target Response: -----------------")
	resBytes, err := httputil.DumpResponse(res, false)
	if err != nil {
		log.Fatal("HttpsConnector.Process() DumpResponse() : ", err)
	}
	fmt.Println(string(resBytes))
	return nil
}

func NewHttpsConnector() (*HttpsConnector, error) {

	rootCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Failed to read system certificates: %v", err)
		return nil, err
	}

	if rootCertPool == nil {
		log.Fatal("Failed to read system certificates: rootCertPool == nil")
		return nil, errors.New("Failed to read system certificates: rootCertPool == nil")
	}

	tlsConfig := &tls.Config{
		RootCAs: rootCertPool,
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &HttpsConnector{client: &http.Client{Transport: tr}}, nil
}
