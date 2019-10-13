package wrapper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"strings"

	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

func GetConf(filename string) (map[string]string, error) {
	conf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	kvs := map[string]string{}
	for _, line := range strings.Split(string(conf), "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] != '#' {
			kv := strings.SplitN(line, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("incorrectly formatted configuration file '%s'", filename)
			}
			kvs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return kvs, nil
}

func GetClusterConf() (map[string]string, error) {
	return GetConf(paths.ClusterConfPath)
}

func GetLocalConf() (map[string]string, error) {
	return GetConf(paths.LocalConfPath)
}

func GetAPIServer() (string, error) {
	kvs, err := GetClusterConf()
	if err != nil {
		return "", err
	}
	apiserver, found := kvs["APISERVER"]
	if !found {
		return "", errors.New("could not find key 'APISERVER'")
	}
	return apiserver, nil
}

func GenerateKubeConfigForAPIServerToFile(apiserver, keypath, certpath, filename string) error {
	config := api.Config{
		CurrentContext: "hyades",
		Clusters: map[string]*api.Cluster{
			"hyades-cluster": {
				// just use one API server for now
				// TODO: BETTER HIGH-AVAILABILITY
				Server:               apiserver,
				CertificateAuthority: paths.KubernetesCAPath,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"kubelet-auth": {
				ClientCertificate: certpath,
				ClientKey:         keypath,
			},
		},
		Contexts: map[string]*api.Context{
			"hyades": {
				Cluster:  "hyades-cluster",
				AuthInfo: "kubelet-auth",
			},
		},
	}
	return clientcmd.WriteToFile(config, filename)
}

func GenerateKubeConfigToFile(keypath, certpath, filename string) error {
	apiserver, err := GetAPIServer()
	if err != nil {
		return err
	}
	return GenerateKubeConfigForAPIServerToFile(apiserver, keypath, certpath, filename)
}
