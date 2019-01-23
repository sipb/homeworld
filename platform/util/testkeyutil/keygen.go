package testkeyutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"github.com/sipb/homeworld/platform/util/wraputil"
	"math/big"
	"net"
	"testing"
	"time"
)

func GenerateTLSRootForTests(t *testing.T, commonname string, dns []string, ips []net.IP) (*rsa.PrivateKey, *x509.Certificate) {
	return GenerateTLSKeypairForTests(t, commonname, dns, ips, nil, nil)
}

// based on https://stackoverflow.com/questions/33932221/golang-marshal-pkcs8-private-key
func marshalpkcs8(key *rsa.PrivateKey) ([]byte, error) {
	return asn1.Marshal(struct {
		Version             int
		PrivateKeyAlgorithm []asn1.ObjectIdentifier
		PrivateKey          []byte
	}{
		Version:             0,
		PrivateKeyAlgorithm: []asn1.ObjectIdentifier{{1, 2, 840, 113549, 1, 1, 1}}, // pkcs1 OID
		PrivateKey:          x509.MarshalPKCS1PrivateKey(key),
	})
}

func GenerateTLSRootPEMsForTests(t *testing.T, commonname string, dns []string, ips []net.IP) (key []byte, keypkcs8 []byte, cert []byte) {
	keyd, certd := GenerateTLSRootForTests(t, commonname, dns, ips)
	pkcs8key, err := marshalpkcs8(keyd)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keyd)}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8key}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certd.Raw})
}

func GenerateTLSKeypairForTests(t *testing.T, commonname string, dns []string, ips []net.IP, parent *x509.Certificate, parentkey *rsa.PrivateKey) (*rsa.PrivateKey, *x509.Certificate) {
	return GenerateTLSKeypairForTests_WithTime(t, commonname, dns, ips, parent, parentkey, time.Now(), time.Hour)
}

func GenerateTLSKeypairForTests_WithTime(t *testing.T, commonname string, dns []string, ips []net.IP, parent *x509.Certificate, parentkey *rsa.PrivateKey, issueat time.Time, duration time.Duration) (*rsa.PrivateKey, *x509.Certificate) {
	key, err := rsa.GenerateKey(rand.Reader, 512) // NOTE: this is LAUGHABLY SMALL! do not attempt to use this in production.
	if err != nil {
		t.Fatal("Could not generate TLS keypair: " + err.Error())
	}

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
		IsCA:       true,
		MaxPathLen: 1,

		SerialNumber: serialNumber,

		NotBefore: issueat,
		NotAfter:  issueat.Add(duration),

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

func GenerateTLSKeypairPEMsForTests(t *testing.T, commonname string, dns []string, ips []net.IP, parent []byte, parentkey []byte) (key []byte, keypkcs8 []byte, cert []byte) {
	parentdec, err := wraputil.LoadX509CertFromPEM(parent)
	if err != nil {
		t.Fatal(err)
	}
	parentkeydec, err := wraputil.LoadRSAKeyFromPEM(parentkey)
	if err != nil {
		t.Fatal(err)
	}
	keyd, certd := GenerateTLSKeypairForTests(t, commonname, dns, ips, parentdec, parentkeydec)
	pkcs8key, err := marshalpkcs8(keyd)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(keyd)}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8key}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certd.Raw})
}
