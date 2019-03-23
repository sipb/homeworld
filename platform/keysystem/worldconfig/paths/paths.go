package paths

import (
	"io/ioutil"
	"strings"
)

func GetKeyserver() (string, error) {
	keyserver, err := ioutil.ReadFile("/etc/homeworld/config/keyserver.domain")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(keyserver)) + ":20557", nil
}
