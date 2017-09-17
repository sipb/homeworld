package main

import (
	"log"
	"os"
	"keyserver/config"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
	"time"
	"keycommon/server"
	"crypto/tls"
	"util/wraputil"
	"keycommon/reqtarget"
)

func main() {
	logger := log.New(os.Stderr, "[keyinitadmit] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 4 {
		logger.Fatal("usage: keyinitadmit <keyserver-config> <server> <bootstrap-api>\n  runs on the keyserver; requests a bootstrap token using privileged access")
	}
	principal := os.Args[2]
	bootstrap_api := os.Args[3]
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
	certdata, err := ctx.AuthenticationAuthority.Sign(string(csr), false, time.Minute * 10, principal, nil)
	if err != nil {
		logger.Fatal(err)
	}
	ks, err := server.NewKeyserver(ctx.ServerTLS.GetPublicKey(), principal + ":20557")
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
