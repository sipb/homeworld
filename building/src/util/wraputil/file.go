package wraputil

import (
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
)

func LoadRSAKeyFromPath(path string) (*rsa.PrivateKey, error) {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadRSAKeyFromPEM(cert)
}

func LoadX509FromPath(path string) (*x509.Certificate, error) {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadX509CertFromPEM(cert)
}
