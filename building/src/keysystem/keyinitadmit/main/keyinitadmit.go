package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"keysystem/api/reqtarget"
	"keysystem/api/server"
	"keysystem/keyserver/config"
	"log"
	"os"
	"time"
	"util/wraputil"
)

func main() {
	logger := log.New(os.Stderr, "[keyinitadmit] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 5 {
		logger.Fatal("usage: keyinitadmit <keyserver-config> <server> <principal> <bootstrap-api>\n  runs on the keyserver; requests a bootstrap token using privileged access")
	}
	server_name := os.Args[2]
	principal := os.Args[3]
	bootstrap_api := os.Args[4]
	ctx, err := config.LoadConfig(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Fatal(err)
	}
	csrder, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{SignatureAlgorithm: x509.SHA256WithRSA}, privkey)
	if err != nil {
		logger.Fatal(err)
	}
	csr := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrder})
	certdata, err := ctx.AuthenticationAuthority.Sign(string(csr), false, time.Minute*10, server_name, nil)
	if err != nil {
		logger.Fatal(err)
	}
	ks, err := server.NewKeyserver(ctx.ServerTLS.GetPublicKey(), server_name+":20557")
	if err != nil {
		logger.Fatal(err)
	}
	cert, err := wraputil.LoadX509CertFromPEM([]byte(certdata))
	if err != nil {
		logger.Fatal(err)
	}
	rt, err := ks.AuthenticateWithCert(tls.Certificate{PrivateKey: privkey, Certificate: [][]byte{cert.Raw}})
	if err != nil {
		logger.Fatal(err)
	}
	token, err := reqtarget.SendRequest(rt, bootstrap_api, principal)
	if err != nil {
		logger.Fatal(err)
	}
	_, err = os.Stdout.WriteString(token)
	if err != nil {
		logger.Fatal(err)
	}
}
