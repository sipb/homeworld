package keycommon

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"keycommon/server"
	"keycommon/reqtarget"
)

type configtype struct {
	AuthorityPath string
	Keyserver     string
	KeyPath       string
	CertPath      string
}

func LoadKeyserver(configpath string) (*server.Keyserver, configtype, error) {
	config := configtype{}
	configdata, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While loading configuration: %s", err)
	}
	err = yaml.Unmarshal(configdata, &config)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While decoding configuration: %s", err)
	}
	authoritydata, err := ioutil.ReadFile(config.AuthorityPath)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While loading authority: %s", err)
	}
	ks, err := server.NewKeyserver(authoritydata, config.Keyserver)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While preparing setup: %s", err)
	}
	return ks, config, nil
}

func LoadKeyserverWithCert(configpath string) (*server.Keyserver, reqtarget.RequestTarget, error) {
	k, config, err := LoadKeyserver(configpath)
	if err != nil {
		return nil, nil, err
	}
	if config.CertPath == "" || config.KeyPath == "" {
		return nil, nil, fmt.Errorf("While preparing authentication: expected non-empty path.")
	}
	keypair, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("While loading keypair: %s", err)
	}
	rt, err := k.AuthenticateWithCert(keypair)
	if err != nil {
		return nil, nil, fmt.Errorf("While preparing authentication: %s", err)
	}
	return k, rt, nil
}
