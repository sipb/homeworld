package keycommon

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"fmt"
	"crypto/tls"
)

type configtype struct {
	authority_path string
	hostname string
	key_path string
	cert_path string
}

func LoadKeyserver(configpath string) (*Keyserver, configtype, error) {
	config := configtype{}
	configdata, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While loading configuration: %s", err)
	}
	err = yaml.UnmarshalStrict(configdata, &config)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While decoding configuration: %s", err)
	}
	authoritydata, err := ioutil.ReadFile(config.authority_path)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While loading authority: %s", err)
	}
	ks, err := NewKeyserver(authoritydata, config.hostname)
	if err != nil {
		return nil, configtype{}, fmt.Errorf("While preparing setup: %s", err)
	}
	return ks, config, nil
}

func LoadKeyserverWithCert(configpath string) (*Keyserver, RequestTarget, error) {
	k, config, err := LoadKeyserver(configpath)
	if err != nil {
		return nil, nil, err
	}
	if config.cert_path == "" || config.key_path == "" {
		return nil, nil, fmt.Errorf("While preparing authentication: expected non-empty path.")
	}
	keypair, err := tls.LoadX509KeyPair(config.cert_path, config.key_path)
	if err != nil {
		return nil, nil, fmt.Errorf("While loading keypair: %s", err)
	}
	rt, err := k.AuthenticateWithCert(keypair)
	if err != nil {
		return nil, nil, fmt.Errorf("While preparing authentication: %s", err)
	}
	return k, rt, nil
}
