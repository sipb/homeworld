package paths

import (
	"io/ioutil"
	"strings"
)

const KeyserverTLSCert = "/etc/homeworld/keyclient/keyservertls.pem"
const GrantingKeyPath = "/etc/homeworld/keyclient/granting.key"
const GrantingCertPath = "/etc/homeworld/keyclient/granting.pem"
const BootstrapTokenPath = "/etc/homeworld/keyclient/bootstrap.token"
const SpireSetupPath = "/etc/homeworld/config/setup.yaml"

const ClusterConfPath = "/etc/homeworld/config/cluster.conf"
const LocalConfPath = "/etc/homeworld/config/local.conf"

const KubernetesCAPath = "/etc/homeworld/authorities/kubernetes.pem"
const KubernetesMasterKey = "/etc/homeworld/keys/kubernetes-master.key"
const KubernetesMasterCert = "/etc/homeworld/keys/kubernetes-master.pem"
const KubernetesWorkerKey = "/etc/homeworld/keys/kubernetes-worker.key"
const KubernetesWorkerCert = "/etc/homeworld/keys/kubernetes-worker.pem"

func GetKeyserver() (string, error) {
	keyserver, err := ioutil.ReadFile("/etc/homeworld/config/keyserver.domain")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(keyserver)) + ":20557", nil
}
