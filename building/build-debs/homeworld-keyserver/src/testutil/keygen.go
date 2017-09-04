package testutil

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/rand"
	"time"
	"math/big"
	"crypto/x509/pkix"
	"net"
	"testing"
)

func GenerateTLSRootForTests(t *testing.T, commonname string, dns []string, ips []net.IP) (*rsa.PrivateKey, *x509.Certificate) {
	return GenerateTLSKeypairForTests(t, commonname, dns, ips, nil, nil)
}

func GenerateTLSKeypairForTests(t *testing.T, commonname string, dns []string, ips []net.IP, parent *x509.Certificate, parentkey *rsa.PrivateKey) (*rsa.PrivateKey, *x509.Certificate) {
	key, err := rsa.GenerateKey(rand.Reader, 512) // NOTE: this is LAUGHABLY SMALL! do not attempt to use this in production.
	if err != nil {
		t.Fatal("Could not generate TLS keypair: " + err.Error())
	}

	issue_at := time.Now()

	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		t.Fatal("Could not generate TLS keypair: " + err.Error())
	}

	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}

	certTemplate := &x509.Certificate{
		SignatureAlgorithm: x509.SHA256WithRSA,

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: extKeyUsage,

		BasicConstraintsValid: true,
		IsCA:           true,
		MaxPathLen:     1,

		SerialNumber: serialNumber,

		NotBefore: issue_at,
		NotAfter:  issue_at.Add(time.Hour),

		Subject:     pkix.Name{CommonName: commonname},
		DNSNames:    dns,
		IPAddresses: ips,
	}

	if parent == nil {
		parent = certTemplate
		parentkey = key
	}

	signed_cert, err := x509.CreateCertificate(rand.Reader, certTemplate, parent, key.Public(), parentkey)
	if err != nil {
		t.Fatal("Could not generate TLS keypair: " + err.Error())
	}
	cert, err := x509.ParseCertificate(signed_cert)
	if err != nil {
		t.Fatal("Could not generate TLS keypair: " + err.Error())
	}
	return key, cert
}
