package connectors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net/http"
)

type HttpsConnector struct {
	client *http.Client
}

func (usc *HttpsConnector) Process(ctx context.Context, req *http.Request, res *http.Response) error {
	trgtRes, err := usc.client.Do(req)
	if err != nil {
		log.Fatal("Err: Faild to Fire Req to Target through: HttpsConnector.Process() : ", err)
		return err
	}

	*res = *trgtRes
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
