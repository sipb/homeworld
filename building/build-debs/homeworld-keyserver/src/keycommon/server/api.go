package server

import (
	"crypto/x509"
	"fmt"
	"wraputil"
)

type Keyserver struct {
	endpoint ServerEndpoint
}

func NewKeyserver(authority []byte, hostname string) (*Keyserver, error) {
	cert, err := wraputil.LoadX509CertFromPEM(authority)
	if err != nil {
		return nil, fmt.Errorf("While parsing authority certificate: %s", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)

	endpoint, err := NewServerEndpoint("https://" + hostname + ":20557/", pool)
	if err != nil {
		return nil, err
	}
	return &Keyserver{endpoint: endpoint}, nil
}

func (k *Keyserver) GetStatic(staticname string) ([]byte, error) {
	if staticname == "" {
		return nil, fmt.Errorf("Static filename is empty.")
	}
	return k.endpoint.Get("/static/" + staticname)
}

func (k *Keyserver) GetPubkey(authorityname string) ([]byte, error) {
	if authorityname == "" {
		return nil, fmt.Errorf("Authority name is empty.")
	}
	return k.endpoint.Get("/pub/" + authorityname)
}
