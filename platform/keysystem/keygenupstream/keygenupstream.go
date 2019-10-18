package main

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keygen"
	"github.com/sipb/homeworld/platform/util/certutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func Generate(cakeyfile, cacertfile, email, userkeyfile, usercertfile string, lifespan time.Duration) error {
	// generate certificate authority
	cakey, cakeybytes, err := certutil.GenerateRSA(keygen.AuthorityBits)
	if err != nil {
		return err
	}
	cacertbytes, err := keygen.GenerateTLSSelfSignedCert(cakey, "upstream-local")
	if err != nil {
		return err
	}
	cacert, err := wraputil.LoadX509CertFromPEM(cacertbytes)
	if err != nil {
		return err
	}

	// generate user key and certificate
	userkey, userkeybytes, err := certutil.GenerateRSA(keygen.AuthorityBits)

	issueAt := time.Now()

	certTemplate := &x509.Certificate{
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  false,
		MaxPathLen:            0,
		MaxPathLenZero:        true,

		NotBefore: issueAt,
		NotAfter:  issueAt.Add(lifespan),

		Subject: pkix.Name{
			CommonName: "simulated user " + email,
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  certutil.OIDEmailAddress(),
					Value: email,
				},
			},
		},
	}

	usercertbytes, err := certutil.FinishCertificate(certTemplate, cacert, userkey.Public(), cakey)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(cakeyfile, cakeybytes, os.FileMode(0600))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(cacertfile, cacertbytes, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(userkeyfile, userkeybytes, os.FileMode(0600))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(usercertfile, usercertbytes, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	logger := log.New(os.Stderr, "[keygenupstream] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 7 {
		logger.Fatalln("usage: keygenupstream <ca-key-out> <ca-cert-out> <email> <user-key-out> <user-cert-out> <lifespan>\n  generates an upstream authority and client cert directly")
	}
	lifespan, err := time.ParseDuration(os.Args[6])
	if err != nil {
		logger.Fatalln(err)
	}
	err = Generate(os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], lifespan)
	if err != nil {
		logger.Fatalln(err)
	}
	logger.Print("done generating fake upstream info.")
}
