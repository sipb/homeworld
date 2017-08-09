package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type ConfigAuthority struct {
	Type string
	Key  string
	Cert string
}

type ConfigAccount struct {
	Principal         string
	Group             string
	LimitIP           bool `yaml:"limit-ip"`
	DisableDirectAuth bool `yaml:"disable-direct-auth"`
	Metadata          map[string]string
}

type ConfigGrant struct {
	Group        string
	Privilege    string
	Scope        string
	Authority    string
	IsHost       string
	Lifespan     string
	CommonName   string   `yaml:"common-name"`
	AllowedNames []string `yaml:"allowed-names"`
	Contents     string
}

type ConfigGroup struct {
	Inherit string
}

type ConfigStatic string

type Config struct {
	AuthorityDir            string
	StaticDir               string
	AuthenticationAuthority string `yaml:"authentication-authority"`
	ServerTLS               string
	Authorities             map[string]ConfigAuthority
	StaticFiles             []ConfigStatic
	Accounts                []ConfigAccount
	Groups                  map[string]ConfigGroup
	Grants                  map[string]ConfigGrant
}

func parseConfigFromBytes(data []byte) (*Config, error) {
	config := new(Config)
	err := yaml.UnmarshalStrict(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func LoadConfigFromBytes(data []byte) (*Context, error) {
	config, err := parseConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	return config.Compile()
}

func LoadConfig(filename string) (*Context, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return LoadConfigFromBytes(contents)
}
