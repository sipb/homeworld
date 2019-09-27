package worldconfig

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
)

const Supervisor = "supervisor"
const Master = "master"
const Worker = "worker"

type SpireNode struct {
	Hostname string
	IP       string
	netIP    net.IP
	setup    *SpireSetup
	Kind     string
}

func (s *SpireNode) IsSupervisor() bool {
	return s.Kind == Supervisor
}
func (s *SpireNode) IsMaster() bool {
	return s.Kind == Master
}
func (s *SpireNode) IsWorker() bool {
	return s.Kind == Worker
}

func (s *SpireNode) DNS() string {
	if s.setup == nil {
		panic("uninitialized")
	}
	return s.Hostname + "." + s.setup.Cluster.ExternalDomain
}

func (s *SpireNode) NetIP() net.IP {
	if s.netIP == nil {
		panic("uninitialized")
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
	supervisor *SpireNode
	RootAdmins []string `yaml:"root-admins"`
}

func (s *SpireSetup) Supervisor() *SpireNode {
	if s.supervisor == nil {
		panic("uninitialized")
	}
	return s.supervisor
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
	supervisors := 0
	for _, node := range setup.Nodes {
		if !(node.IsSupervisor() || node.IsMaster() || node.IsWorker()) {
			return nil, fmt.Errorf("unrecognized kind of node: %s", node.Kind)
		}
		node.netIP = net.ParseIP(node.IP)
		if node.netIP == nil {
			return nil, fmt.Errorf("could not parse IP: %s", node.IP)
		}
		node.setup = setup
		if node.IsSupervisor() {
			setup.supervisor = node
			supervisors++
		}
	}
	if supervisors != 1 {
		return nil, fmt.Errorf("expected exactly 1 declared supervisor, not %d", supervisors)
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
