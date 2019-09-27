package keyapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/pkg/errors"
	"github.com/sipb/homeworld/platform/keysystem/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/operation"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

const TemporaryCertificateBits = keygen.AuthorityBits

func apiToHTTP(ks Keyserver, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/apirequest", func(writer http.ResponseWriter, request *http.Request) {
		err := ks.HandleAPIRequest(writer, request)
		if err != nil {
			logger.Printf("API request failed with error: %s", err)
			if _, ok := err.(*operation.OperationForbiddenError); ok {
				http.Error(writer, "Particular operation forbidden.", http.StatusForbidden)
			} else {
				http.Error(writer, "Request processing failed. See server logs for details.", http.StatusBadRequest)
			}
		}
	})

	mux.HandleFunc("/pub/", func(writer http.ResponseWriter, request *http.Request) {
		err := ks.HandlePubRequest(writer, request.URL.Path[len("/pub/"):])
		if err != nil {
			logger.Printf("Public key request failed with error: %s", err)
			http.Error(writer, "Request processing failed: "+err.Error(), http.StatusNotFound)
		}
	})

	mux.HandleFunc("/static/", func(writer http.ResponseWriter, request *http.Request) {
		err := ks.HandleStaticRequest(writer, request.URL.Path[len("/static/"):])
		if err != nil {
			logger.Printf("Static request failed with error: %s", err)
			http.Error(writer, "Request processing failed: "+err.Error(), http.StatusNotFound)
		}
	})

	return mux
}

func generateServerCertificate(ctx *config.Context) (tls.Certificate, error) {
	// TODO: refactor out the key generation pattern
	privateKey, err := rsa.GenerateKey(rand.Reader, TemporaryCertificateBits)
	if err != nil {
		return tls.Certificate{}, err
	}
	contents := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	csr, err := csrutil.BuildTLSCSR(contents)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "while generating CSR")
	}
	// TODO: figure out the right period of time for this certificate to be valid
	cert, err := ctx.ClusterCA.Sign(string(csr), true, time.Hour*24, "keyserver-autogen-tls", []string{ctx.KeyserverDNS})
	if err != nil {
		return tls.Certificate{}, err
	}

	pair, err := tls.X509KeyPair([]byte(cert), contents)
	if err != nil {
		return tls.Certificate{}, err
	}
	return pair, nil
}

func LoadConfiguredKeyserver(logger *log.Logger) (Keyserver, error) {
	ctx, err := worldconfig.GenerateConfig()
	if err != nil {
		return nil, err
	}

	serverCert, err := generateServerCertificate(ctx)
	if err != nil {
		return nil, err
	}

	return &ConfiguredKeyserver{Context: ctx, ServerCert: serverCert, Logger: logger}, nil
}

// addr: ":20557"
func Run(addr string, logger *log.Logger) (func(), chan error, error) {
	ks, err := LoadConfiguredKeyserver(logger)
	if err != nil {
		return nil, nil, err
	}

	server := &http.Server{
		Addr:    addr,
		Handler: apiToHTTP(ks, logger),
		TLSConfig: &tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			ClientCAs:    ks.GetClientCAs(),
			Certificates: []tls.Certificate{ks.GetServerCert()},
			MinVersion:   tls.VersionTLS12,
			NextProtos:   []string{"http/1.1", "h2"},
		},
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, nil, err
	}

	cherr := make(chan error)

	go func() {
		tlsListener := tls.NewListener(ln, server.TLSConfig)
		cherr <- server.Serve(tlsListener)
	}()

	return func() { server.Shutdown(context.Background()) }, cherr, nil
}
