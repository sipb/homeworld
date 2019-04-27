package hostname

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
)

func readConfig(path string) (map[string]string, error) {
	conf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	kvs := map[string]string{}
	for _, line := range strings.Split(string(conf), "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] != '#' {
			kv := strings.SplitN(line, "=", 2)
			if len(kv) != 2 {
				return nil, errors.New("incorrectly formatted local.conf")
			}
			kvs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return kvs, nil
}

func performReload(path string, nac *actloop.NewActionContext) error {
	conf, err := readConfig(path)
	if os.IsNotExist(err) {
		nac.Blocked(err)
		return nil
	} else if err != nil {
		return err
	} else {
		hostname, found := conf["HOST_NODE"]
		if !found {
			return errors.New("no HOST_NODE entry in local.conf")
		}
		currentHostname, err := os.Hostname()
		if err != nil {
			return err
		}
		if hostname == currentHostname {
			return nil
		}
		err = exec.Command("hostnamectl", "set-hostname", hostname).Run()
		if err != nil {
			return err
		}
		nac.NotifyPerformed("reload hostname")
		return nil
	}
}

func ReloadHostnameFrom(path string, nac *actloop.NewActionContext) {
	err := performReload(path, nac)
	if err != nil {
		nac.Errored("reload hostname", err)
	}
}
