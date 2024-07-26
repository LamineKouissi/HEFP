package listeners

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
)

type TLSListener struct {
	address     string
	cnx         context.Context
	server      *http.Server
	tlsCertFile string
	tlsKeyFile  string
}

func (srv *TLSListener) Listen() error {
	log.Printf("TLSServer Listening on %s...", srv.address)
	err := srv.server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
		return err
	}
	return nil
}

func NewTLSListener(cntx context.Context, adrs string, router http.Handler, crtFilePath string, keyFilePath string) (*TLSListener, error) {
	cert, err := tls.LoadX509KeyPair(crtFilePath, keyFilePath)
	if err != nil {
		log.Fatalf("Failed to load X509 key pair: %v", err)
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	srv := &http.Server{
		Addr:      adrs,
		Handler:   router,
		TLSConfig: config,
	}
	return &TLSListener{address: adrs, cnx: cntx, server: srv}, nil
}
