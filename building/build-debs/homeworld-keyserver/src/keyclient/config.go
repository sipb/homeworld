package keyclient

import (
	"keycommon"
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type ConfigDownload struct {
	Type    string
	Name    string
	Path    string
	Refresh string
	Mode    string
}

type ConfigKey struct {
	Name      string
	Type      string
	Key       string
	Cert      string
	API       string
	InAdvance string
}

type Config struct {
	AuthorityPath string
	Keyserver     string
	KeyPath       string
	CertPath      string
	TokenPath     string
	TokenAPI      string
	Downloads     []ConfigDownload
	Keys          []ConfigKey
}

func LoadConfig(configpath string) (*keycommon.Keyserver, Config, error) {
	config := Config{}
	configdata, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, Config{}, fmt.Errorf("While loading configuration: %s", err)
	}
	err = yaml.Unmarshal(configdata, &config)
	if err != nil {
		return nil, Config{}, fmt.Errorf("While decoding configuration: %s", err)
	}
	authoritydata, err := ioutil.ReadFile(config.AuthorityPath)
	if err != nil {
		return nil, Config{}, fmt.Errorf("While loading authority: %s", err)
	}
	ks, err := keycommon.NewKeyserver(authoritydata, config.Keyserver)
	if err != nil {
		return nil, Config{}, fmt.Errorf("While preparing setup: %s", err)
	}
	return ks, config, nil
}
