package paths

import (
	"io/ioutil"
	"strings"
)

const KeyserverTLSCert = "/etc/homeworld/keyclient/keyservertls.pem"
const GrantingKeyPath = "/etc/homeworld/keyclient/granting.key"
const GrantingCertPath = "/etc/homeworld/keyclient/granting.pem"
const BootstrapTokenPath = "/etc/homeworld/keyclient/bootstrap.token"
const KeyserverConfigPath = "/etc/homeworld/config/keyserver.yaml"
const SpireSetupPath = "/etc/homeworld/config/setup.yaml"

const BootstrapKeyserverTokenAPI = "bootstrap-keyinit"
const BootstrapTokenAPI = "renew-keygrant"

func GetKeyserver() (string, error) {
	keyserver, err := ioutil.ReadFile("/etc/homeworld/config/keyserver.domain")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(keyserver)) + ":20557", nil
}
