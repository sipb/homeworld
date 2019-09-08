package server

import (
	"crypto/x509"
	"fmt"
	"github.com/pkg/errors"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

type Keyserver struct {
	endpoint endpoint.ServerEndpoint
}

func NewKeyserver(authority []byte, hostname string) (*Keyserver, error) {
	cert, err := wraputil.LoadX509CertFromPEM(authority)
	if err != nil {
		return nil, errors.Wrap(err, "while parsing authority certificate")
	}

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	// TODO: more robust hostname handling code
	ep, err := endpoint.NewServerEndpoint(fmt.Sprintf("https://%s/", hostname), pool)
	if err != nil {
		return nil, err // should not happen -- the URL provided also satisfies the checked constraints
	}
	return &Keyserver{endpoint: ep}, nil
}

func (k *Keyserver) GetStatic(staticname string) ([]byte, error) {
	if staticname == "" {
		return nil, errors.New("static filename is empty")
	}
	return k.endpoint.Get("/static/" + staticname)
}

func (k *Keyserver) GetPubkey(authorityname string) ([]byte, error) {
	if authorityname == "" {
		return nil, errors.New("authority name is empty")
	}
	return k.endpoint.Get("/pub/" + authorityname)
}
