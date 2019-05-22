package worldconfig

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
)

type SpireNode struct {
	Hostname string
	IP       string
	netIP    net.IP
	Kind     string
}

func (s *SpireNode) NetIP() net.IP {
	if s.netIP == nil {
		panic("IP state inconsistency")
	}
	return s.netIP
}

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
	Nodes      []*SpireNode
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
	// validation steps
	for _, node := range setup.Nodes {
		if node.Kind != "worker" && node.Kind != "supervisor" && node.Kind != "master" {
			return nil, fmt.Errorf("unrecognized kind of node: %s", node.Kind)
		}
		node.netIP = net.ParseIP(node.IP)
		if node.netIP == nil {
			return nil, fmt.Errorf("could not parse IP: %s", node.IP)
		}
	}
	dupcheck := map[string]bool{}
	for _, rootadmin := range setup.RootAdmins {
		if rootadmin == "" {
			return nil, errors.New("invalid root admin name ''")
		}
		if dupcheck[rootadmin] {
			return nil, fmt.Errorf("duplicate root admin: %s", rootadmin)
		}
		dupcheck[rootadmin] = true
	}
	return setup, nil
}
