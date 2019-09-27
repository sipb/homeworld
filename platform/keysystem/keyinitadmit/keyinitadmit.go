package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func main() {
	logger := log.New(os.Stderr, "[keyinitadmit] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 2 {
		logger.Fatal("usage: keyinitadmit <principal>\n  runs on the keyserver; requests a bootstrap token using privileged access")
	}
	principal := os.Args[1]
	ctx, err := worldconfig.GenerateConfig()
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
	certdata, err := ctx.AuthenticationAuthority.Sign(string(csr), false, time.Minute*10, ctx.KeyserverDNS, nil)
	if err != nil {
		logger.Fatal(err)
	}
	ks, err := server.NewKeyserver(ctx.ClusterCA.GetPublicKey(), ctx.KeyserverDNS+":20557")
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
	token, err := reqtarget.SendRequest(rt, paths.BootstrapKeyserverTokenAPI, principal)
	if err != nil {
		logger.Fatal(err)
	}
	_, err = os.Stdout.WriteString(token)
	if err != nil {
		logger.Fatal(err)
	}
}
