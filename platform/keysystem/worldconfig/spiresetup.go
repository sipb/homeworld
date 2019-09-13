package worldconfig

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// format for the setup.yaml that spire uses
type SpireSetup struct {
	Cluster struct {
		ExternalDomain string `yaml:"external-domain"`
		InternalDomain string `yaml:"internal-domain"`
		KerberosRealm  string `yaml:"kerberos-realm"`
	}
	Addresses struct {
		ServiceAPI string `yaml:"service-api"`
	}
	Nodes []struct {
		Hostname string
		IP       string
		Kind     string
	}
	RootAdmins []string `yaml:"root-admins"`
}

func LoadSpireSetup(path string) (*SpireSetup, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	setup := &SpireSetup{}
	err = yaml.Unmarshal(content, setup)
	if err != nil {
		return nil, err
	}
	return setup, nil
}
