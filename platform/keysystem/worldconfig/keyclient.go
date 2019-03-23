package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/config"
)

const (
	BASE       = "base"
	SUPERVISOR = "supervisor"
	MASTER     = "master"
	WORKER     = "worker"
)

func GenerateConfig(keyserver string, variant string) (*config.Config, error) {
	if variant != BASE && variant != SUPERVISOR && variant != MASTER && variant != WORKER {
		return nil, fmt.Errorf("invalid variant: %s", variant)
	}
	conf := &config.Config{
		Keyserver:     keyserver + ":20557",
		AuthorityPath: "/etc/homeworld/keyclient/keyservertls.pem",
		KeyPath:       "/etc/homeworld/keyclient/granting.key",
		CertPath:      "/etc/homeworld/keyclient/granting.pem",
		TokenPath:     "/etc/homeworld/keyclient/bootstrap.token",
		TokenAPI:      "renew-keygrant",
	}
	conf.Downloads = []config.ConfigDownload{
		{
			Type:    "authority",
			Name:    "kubernetes",
			Path:    "/etc/homeworld/authorities/kubernetes.pem",
			Refresh: "24h",
			Mode:    "644",
		},
		{
			Type:    "authority",
			Name:    "clustertls",
			Path:    "/usr/local/share/ca-certificates/extra/cluster.tls.crt",
			Refresh: "24h",
			Mode:    "644",
		},
		{
			Type:    "authority",
			Name:    "ssh-user",
			Path:    "/etc/ssh/ssh_user_ca.pub",
			Refresh: "168h", // allow a week for mistakes to be noticed on this one
			Mode:    "644",
		},
		{
			Type:    "static",
			Name:    "cluster.conf",
			Path:    "/etc/homeworld/config/cluster.conf",
			Refresh: "24h",
			Mode:    "644",
		},
		{
			Type:    "api",
			Name:    "get-local-config",
			Path:    "/etc/homeworld/config/local.conf",
			Refresh: "24h",
			Mode:    "644",
		},
	}
	if variant == MASTER {
		conf.Downloads = append(conf.Downloads,
			config.ConfigDownload{
				Type:    "authority",
				Name:    "serviceaccount",
				Path:    "/etc/homeworld/keys/serviceaccount.pem",
				Refresh: "24h",
				Mode:    "644",
			},
			config.ConfigDownload{
				Type:    "api",
				Name:    "fetch-serviceaccount-key",
				Path:    "/etc/homeworld/keys/serviceaccount.key",
				Refresh: "24h",
				Mode:    "600",
			},
			config.ConfigDownload{
				Type:    "authority",
				Name:    "etcd-client",
				Path:    "/etc/homeworld/authorities/etcd-client.pem",
				Refresh: "24h",
				Mode:    "644",
			},
			config.ConfigDownload{
				Type:    "authority",
				Name:    "etcd-server",
				Path:    "/etc/homeworld/authorities/etcd-server.pem",
				Refresh: "24h",
				Mode:    "644",
			},
		)
	}
	conf.Keys = []config.ConfigKey{
		{
			Name:      "keygranting",
			Type:      "tls",
			Key:       "/etc/homeworld/keyclient/granting.key",
			Cert:      "/etc/homeworld/keyclient/granting.pem",
			API:       "renew-keygrant",
			InAdvance: "336h", // renew two weeks before expiration
		},
		{
			Name:      "ssh-host",
			Type:      "ssh-pubkey",
			Key:       "/etc/ssh/ssh_host_rsa_key.pub",
			Cert:      "/etc/ssh/ssh_host_rsa_cert",
			API:       "grant-ssh-host",
			InAdvance: "168h", // renew one week before expiration
		},
		{
			// for master nodes, worker nodes (both for kubelet), and supervisor nodes (for prometheus)
			Name:      "kube-worker",
			Type:      "tls",
			Key:       "/etc/homeworld/keys/kubernetes-worker.key",
			Cert:      "/etc/homeworld/keys/kubernetes-worker.pem",
			API:       "grant-kubernetes-worker",
			InAdvance: "168h", // renew one week before expiration
		},
	}
	if variant == SUPERVISOR {
		conf.Keys = append(conf.Keys,
			config.ConfigKey{
				Name:      "clustertls",
				Type:      "tls",
				Key:       "/etc/homeworld/ssl/homeworld.private.key",
				Cert:      "/etc/homeworld/ssl/homeworld.private.pem",
				API:       "grant-registry-host",
				InAdvance: "168h", // renew one week before expiration
			},
		)
	} else if variant == MASTER {
		conf.Keys = append(conf.Keys,
			config.ConfigKey{
				Name:      "kube-master",
				Type:      "tls",
				Key:       "/etc/homeworld/keys/kubernetes-master.key",
				Cert:      "/etc/homeworld/keys/kubernetes-master.pem",
				API:       "grant-kubernetes-master",
				InAdvance: "168h", // renew one week before expiration
			},
			config.ConfigKey{
				Name:      "etcd-server",
				Type:      "tls",
				Key:       "/etc/homeworld/keys/etcd-server.key",
				Cert:      "/etc/homeworld/keys/etcd-server.pem",
				API:       "grant-etcd-server",
				InAdvance: "168h", // renew one week before expiration
			},
			config.ConfigKey{
				Name:      "etcd-client",
				Type:      "tls",
				Key:       "/etc/homeworld/keys/etcd-client.key",
				Cert:      "/etc/homeworld/keys/etcd-client.pem",
				API:       "grant-etcd-client",
				InAdvance: "168h", // renew one week before expiration
			},
		)
	}
	return conf, nil
}

func GenerateVariant(keyserver string, variant string) (string, error) {
	conf, err := GenerateConfig(keyserver, variant)
	if err != nil {
		return "", err
	}
	data, err := yaml.Marshal(conf)
	if err != nil {
		return "", err
	}
	output := string(data)
	if variant == BASE {
		output = "# TEMPORARY-KEYCLIENT-CONFIGURATION\n\n" + output
	}
	return output, nil
}

func main() {
	logger := log.New(os.Stderr, "[keyconfgen] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 3 {
		logger.Fatalln("usage: keyconfgen <keyserver.domain> <variant>\n  generates the configuration for a keyclient")
	}
	keyserver := os.Args[1]
	variant := os.Args[2]
	result, err := GenerateVariant(keyserver, variant)
	if err != nil {
		logger.Fatalln(err)
	}
	fmt.Println(result)
}
