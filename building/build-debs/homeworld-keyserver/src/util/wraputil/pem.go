package wraputil

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

func LoadSinglePEMBlock(data []byte, expected_types []string) ([]byte, error) {
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
			break
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

func LoadX509CertFromPEM(certdata []byte) (*x509.Certificate, error) {
	certblock, err := LoadSinglePEMBlock(certdata, []string{"CERTIFICATE"})
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(certblock)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func LoadX509CSRFromPEM(certdata []byte) (*x509.CertificateRequest, error) {
	pemBlock, err := LoadSinglePEMBlock(certdata, []string{"CERTIFICATE REQUEST"})
	if err != nil {
		return nil, err
	}
	csr, err := x509.ParseCertificateRequest(pemBlock)
	if err != nil {
		return nil, err
	}
	return csr, nil
}

func LoadRSAKeyFromPEM(keydata []byte) (*rsa.PrivateKey, error) {
	keyblock, err := LoadSinglePEMBlock(keydata, []string{"RSA PRIVATE KEY", "PRIVATE KEY"})
	if err != nil {
		return nil, err
	}

	privkey, err := x509.ParsePKCS1PrivateKey(keyblock)
	if err == nil {
		return privkey, nil
	}
	tmpkey, err := x509.ParsePKCS8PrivateKey(keyblock)
	if err == nil {
		privkey, ok := tmpkey.(*rsa.PrivateKey)
		if ok {
			return privkey, nil
		} else {
			return nil, errors.New("non-RSA private key found in PKCS#8 block")
		}
	}
	return nil, errors.New("could not load PEM private key as PKCS#1 or PKCS#8")
}
