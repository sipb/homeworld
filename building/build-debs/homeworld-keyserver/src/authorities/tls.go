package authorities

import (
	"net/http"
	"fmt"
	"crypto/x509"
	"time"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"crypto/rand"
	"math/big"
	"crypto/x509/pkix"
	"net"
	"crypto/tls"
	"bytes"
)

type TLSAuthority struct {
	// TODO: also support ECDSA or other newer algorithms
	key         *rsa.PrivateKey
	cert        *x509.Certificate
	certData    []byte
	certEncoded []byte
}

func (t *TLSAuthority) Equal(authority *TLSAuthority) bool {
	return bytes.Equal(t.certData, authority.certData)
}

func loadSinglePEMBlock(data []byte, expected_types []string) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte("-----BEGIN ")) {
		return nil, errors.New("Missing expected PEM header")
	}
	pemBlock, remain := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("Could not parse PEM data")
	}
	found := false
	for _, expected_type := range expected_types {
		if pemBlock.Type == expected_type {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("Found PEM block of type \"%s\" instead of types %s", pemBlock.Type, expected_types)
	}
	if remain != nil && len(remain) > 0 {
		return nil, errors.New("Trailing data found after PEM data")
	}
	return pemBlock.Bytes, nil
}

func LoadTLSAuthority(keydata []byte, pubkeydata []byte) (Authority, error) {
	certblock, err := loadSinglePEMBlock(pubkeydata, []string{"CERTIFICATE"})
	if err != nil {
		return nil, err
	}

	keyblock, err := loadSinglePEMBlock(keydata, []string{"RSA PRIVATE KEY", "PRIVATE KEY"})
	if err != nil {
		return nil, err
	}

	privkey, err := x509.ParsePKCS1PrivateKey(keyblock)
	if err != nil {
		tmpkey, err := x509.ParsePKCS8PrivateKey(keyblock)
		if err != nil {
			return nil, errors.New("Could not load PEM private key as PKCS#1 or PKCS#8")
		}
		tprivkey, ok := tmpkey.(*rsa.PrivateKey)
		if ok {
			privkey = tprivkey
		} else {
			return nil, errors.New("Non-RSA private key found in PKCS#8 block")
		}
	}

	cert, err := x509.ParseCertificate(certblock)
	if err != nil {
		return nil, err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("expected RSA public key in certificate")
	}
	if pub.N.Cmp(privkey.N) != 0 {
		return nil, errors.New("mismatched RSA public and private keys")
	}

	return &TLSAuthority{key: privkey, cert: cert, certData: certblock, certEncoded: pubkeydata}, nil
}

func (t *TLSAuthority) ToCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(t.cert)
	return pool
}

func (t *TLSAuthority) GetPublicKey() []byte {
	return t.certEncoded
}

func (t *TLSAuthority) ToHTTPSCert() tls.Certificate {
	return tls.Certificate{Certificate: [][]byte{t.certData }, PrivateKey: t.key }
}

func (t *TLSAuthority) Verify(request *http.Request) (string, error) {
	if len(request.TLS.VerifiedChains) == 0 || len(request.TLS.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("Client certificate must be present")
	}
	firstCert := request.TLS.VerifiedChains[0][0]
	err := firstCert.CheckSignatureFrom(t.cert) // duplicate effort? probably. do it anyway, so we're certain it's *THIS* authority.
	if err != nil {
		return "", fmt.Errorf("Certificate not valid under this authority: %s", err)
	}
	principal := firstCert.Subject.CommonName
	now := time.Now()
	if now.Before(firstCert.NotBefore) {
		return "", fmt.Errorf("Certificate for /CN=%s is not yet valid", principal)
	}
	if now.After(firstCert.NotAfter) {
		return "", fmt.Errorf("Certificate for /CN=%s has expired", principal)
	}
	return principal, nil
}

func (d *TLSAuthority) Sign(request string, ishost bool, lifespan time.Duration, commonname string, names []string) (string, error) {
	pemBlock, err := loadSinglePEMBlock([]byte(request), []string{"CERTIFICATE REQUEST"})
	if err != nil {
		return "", err
	}
	csr, err := x509.ParseCertificateRequest(pemBlock)
	if err != nil {
		return "", err
	}
	err = csr.CheckSignature()
	if err != nil {
		return "", err
	}

	issue_at := time.Now()

	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return "", err
	}

	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	if ishost {
		extKeyUsage = append(extKeyUsage, x509.ExtKeyUsageServerAuth)
	}

	dns_names, IPs := partitionDNSNamesAndIPs(names)

	certTemplate := x509.Certificate{
		SignatureAlgorithm: x509.SHA256WithRSA,

		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: extKeyUsage,

		BasicConstraintsValid: true,
		IsCA:                  false,
		SerialNumber:          serialNumber,

		NotBefore: issue_at,
		NotAfter:  issue_at.Add(lifespan),

		Subject:     pkix.Name{CommonName: commonname},
		DNSNames:    dns_names,
		IPAddresses: IPs,
	}

	signed_cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, d.cert, csr.PublicKey, d.key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: signed_cert})), nil
}

func partitionDNSNamesAndIPs(names []string) ([]string, []net.IP) {
	dnses := make([]string, 0)
	ips := make([]net.IP, 0)
	for _, name := range names {
		ip := net.ParseIP(name)
		if ip == nil {
			dnses = append(dnses, name)
		} else {
			ips = append(ips, ip)
		}
	}
	return dnses, ips
}
