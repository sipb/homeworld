package api

import (
	"crypto/tls"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
)

const ConfigPath = "/etc/homeworld/config/keyclient.yaml"

type Config struct {
	AuthorityPath string
	Keyserver     string
	KeyPath       string
	CertPath      string
}

func LoadDefaultKeyserver() (*server.Keyserver, Config, error) {
	config := Config{}
	configdata, err := ioutil.ReadFile(ConfigPath)
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "while loading configuration")
	}
	err = yaml.Unmarshal(configdata, &config)
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "while decoding configuration")
	}
	authoritydata, err := ioutil.ReadFile(config.AuthorityPath)
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "while loading authority")
	}
	ks, err := server.NewKeyserver(authoritydata, config.Keyserver)
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "while preparing setup")
	}
	return ks, config, nil
}

func LoadDefaultKeyserverWithCert() (*server.Keyserver, reqtarget.RequestTarget, error) {
	k, config, err := LoadDefaultKeyserver()
	if err != nil {
		return nil, nil, err
	}
	if config.CertPath == "" || config.KeyPath == "" {
		return nil, nil, errors.New("while preparing authentication: expected non-empty path")
	}
	keypair, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "while loading keypair")
	}
	rt, err := k.AuthenticateWithCert(keypair) // note: no actual way to make this fail in practice
	if err != nil {
		return nil, nil, errors.Wrap(err, "while preparing authentication")
	}
	return k, rt, nil
}
