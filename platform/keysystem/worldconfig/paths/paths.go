package paths

import (
	"io/ioutil"
	"strings"
)

const KeyserverTLSCert = "/etc/homeworld/keyclient/keyservertls.pem"
const GrantingKeyPath = "/etc/homeworld/keyclient/granting.key"
const GrantingCertPath = "/etc/homeworld/keyclient/granting.pem"
const SpireSetupPath = "/etc/homeworld/config/setup.yaml"

func GetKeyserver() (string, error) {
	keyserver, err := ioutil.ReadFile("/etc/homeworld/config/keyserver.domain")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(keyserver)) + ":20557", nil
}
