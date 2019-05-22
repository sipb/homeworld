package api

import (
	"crypto/tls"
	"github.com/pkg/errors"
	"io/ioutil"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

func LoadDefaultKeyserver() (*server.Keyserver, error) {
	authoritydata, err := ioutil.ReadFile(paths.KeyserverTLSCert)
	if err != nil {
		return nil, errors.Wrap(err, "while loading authority")
	}
	keyserver, err := paths.GetKeyserver()
	if err != nil {
		return nil, errors.Wrap(err, "while determining keyserver")
	}
	return server.NewKeyserver(authoritydata, keyserver)
}

func LoadDefaultKeyserverWithCert() (*server.Keyserver, reqtarget.RequestTarget, error) {
	k, err := LoadDefaultKeyserver()
	if err != nil {
		return nil, nil, err
	}
	keypair, err := tls.LoadX509KeyPair(paths.GrantingCertPath, paths.GrantingKeyPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "while loading keypair")
	}
	rt, err := k.AuthenticateWithCert(keypair) // note: no actual way to make this fail in practice
	if err != nil {
		return nil, nil, errors.Wrap(err, "while preparing authentication")
	}
	return k, rt, nil
}
