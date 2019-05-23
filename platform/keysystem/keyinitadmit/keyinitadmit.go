package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/pkg/errors"
	"log"
	"os"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func BypassCertificate(ctx *config.Context) (tls.Certificate, error) {
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	csrder, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{SignatureAlgorithm: x509.SHA256WithRSA}, privkey)
	if err != nil {
		return tls.Certificate{}, err
	}
	csr := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrder})
	certdata, err := ctx.AuthenticationAuthority.Sign(string(csr), false, time.Minute*10, ctx.KeyserverDNS, nil)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert, err := wraputil.LoadX509CertFromPEM([]byte(certdata))
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{PrivateKey: privkey, Certificate: [][]byte{cert.Raw}}, nil
}

func AuthLocalBypass() (reqtarget.RequestTarget, error) {
	ctx, err := worldconfig.GenerateConfig()
	if err != nil {
		return nil, err
	}
	ks, err := server.NewKeyserver(ctx.ClusterCA.GetPublicKey(), ctx.KeyserverDNS+":20557")
	if err != nil {
		return nil, err
	}
	cert, err := BypassCertificate(ctx)
	if err != nil {
		return nil, err
	}
	rt, err := ks.AuthenticateWithCert(cert)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func Approve(principal string, fingerprint string) error {
	approval := admit.AdmitApproval{
		Principal:   principal,
		Fingerprint: fingerprint,
	}
	data, err := json.Marshal(approval)
	if err != nil {
		return err
	}
	rt, err := AuthLocalBypass()
	if err != nil {
		return err
	}
	response, err := reqtarget.SendRequest(rt, worldconfig.ApproveAdmitAPI, string(data))
	if err != nil {
		return err
	}
	if response != "" {
		return errors.New("expected empty response from approval API")
	}
	return nil
}

func ListRequests() (map[string]admit.AdmitState, error) {
	rt, err := AuthLocalBypass()
	if err != nil {
		return nil, err
	}
	response, err := reqtarget.SendRequest(rt, worldconfig.ApproveAdmitAPI, "")
	if err != nil {
		return nil, err
	}
	var admittable map[string]admit.AdmitState
	err = json.Unmarshal([]byte(response), &admittable)
	if err != nil {
		return nil, err
	}
	return admittable, nil
}

func main() {
	logger := log.New(os.Stderr, "[keyinitadmit] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 1 || len(os.Args) > 3 {
		logger.Fatal("expected zero to two arguments: [principal [fingerprint]] OR option --supervisor")
	}
	if len(os.Args) == 2 && os.Args[1] == "--supervisor" {
		conf, err := worldconfig.LoadSpireSetup(paths.SpireSetupPath)
		if err != nil {
			logger.Fatal(err)
		}
		principal := conf.Supervisor().DNS()
		fingerprint, err := admit.LoadFingerprint(paths.GrantingKeyPath)
		if err != nil {
			logger.Fatal(err)
		}
		err = Approve(principal, fingerprint)
		if err != nil {
			logger.Fatal(err)
		}
	} else if len(os.Args) == 1 {
		requests, err := ListRequests()
		if err != nil {
			logger.Fatal(err)
		}
		err = admit.PrintRequests(requests, os.Stdout)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		principal := os.Args[1]
		fingerprint := os.Args[2]
		err := Approve(principal, fingerprint)
		if err != nil {
			logger.Fatal(err)
		}
	}
}
