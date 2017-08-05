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
)

type TLSAuthority struct {
	// TODO: also support ECDSA or other newer algorithms
	key  *rsa.PrivateKey
	cert *x509.Certificate
	certData []byte
}

func loadSinglePEMBlock(data []byte, expected_type string) ([]byte, error) {
	pemBlock, remain := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("Could not parse PEM data")
	}
	if pemBlock.Type != expected_type {
		return nil, fmt.Errorf("Found PEM block of type \"%s\" instead of type \"%s\"", pemBlock.Type, expected_type)
	}
	if remain != nil && len(remain) > 0 {
		return nil, errors.New("Trailing data found after PEM data")
	}
	return pemBlock.Bytes, nil
}

func LoadTLSAuthority(keydata []byte, pubkeydata []byte) (Authority, error) {
	certblock, err := loadSinglePEMBlock(pubkeydata, "CERTIFICATE")
	if err != nil {
		return nil, err
	}

	keyblock, err := loadSinglePEMBlock(pubkeydata, "RSA PRIVATE KEY")
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

	return &TLSAuthority{key: privkey, cert: cert, certData: certblock}, nil
}

func (t *TLSAuthority) Register(pool *x509.CertPool) {
	pool.AddCert(t.cert)
}

func (t *TLSAuthority) GetPublicKey() []byte {
	return t.certData
}

func (t *TLSAuthority) ToHTTPSCert() tls.Certificate {
	return tls.Certificate{Certificate: [][]byte { t.certData }, PrivateKey: t.key }
}

func (t *TLSAuthority) Verify(request *http.Request) (string, error) {
	if len(request.TLS.VerifiedChains) == 0 || len(request.TLS.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("Valid certificate required for generating bootstrap tokens")
	}
	firstCert := request.TLS.VerifiedChains[0][0]
	err := firstCert.CheckSignatureFrom(t.cert) // duplicate effort? probably. do it anyway, so we're certain it's *THIS* authority.
	if err != nil {
		return "", err
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
	pemBlock, err := loadSinglePEMBlock([]byte(request), "CERTIFICATE REQUEST")
	if err != nil {
		return nil, err
	}
	csr, err := x509.ParseCertificateRequest(pemBlock)
	if err != nil {
		return nil, err
	}
	err = csr.CheckSignature()
	if err != nil {
		return nil, err
	}

	issue_at := time.Now()

	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return nil, err
	}

	extKeyUsage := x509.ExtKeyUsageClientAuth
	if ishost {
		extKeyUsage |= x509.ExtKeyUsageServerAuth
	}

	dns_names, IPs := partitionDNSNamesAndIPs(names)

	certTemplate := x509.Certificate{
		PublicKey:          csr.PublicKey,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{extKeyUsage},

		BasicConstraintsValid: true,
		IsCA:                  false,
		MaxPathLen:            1, // TODO: figure out the correct value for this
		SerialNumber:          serialNumber,

		NotBefore: issue_at,
		NotAfter:  issue_at.Add(lifespan),

		Subject:     pkix.Name{CommonName: commonname},
		DNSNames:    dns_names,
		IPAddresses: IPs,
	}

	signed_cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, d.cert, d.cert.PublicKey, d.key)
	if err != nil {
		return nil, err
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
