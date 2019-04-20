package worldconfig

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/bootstrap"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/download"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keyreq"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/config"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

const (
	BASE       = "base"
	SUPERVISOR = "supervisor"
	MASTER     = "master"
	WORKER     = "worker"
)

func getLocalConf() (map[string]string, error) {
	conf, err := ioutil.ReadFile("/etc/homeworld/config/local.conf")
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

func GetVariant() (string, error) {
	kvs, err := getLocalConf()
	// if no local.conf available, default to "base" mode
	if os.IsNotExist(err) {
		return "base", nil
	}
	if err != nil {
		return "", err
	}
	kind, found := kvs["KIND"]
	if !found {
		return "", errors.New("could not find key 'KIND'")
	}
	return kind, nil
}

func (b *ActionBuilder) DefaultDownloads(variant string) {
	b.Download(config.ConfigDownload{
		Type:    "authority",
		Name:    "kubernetes",
		Path:    "/etc/homeworld/authorities/kubernetes.pem",
		Refresh: "24h",
		Mode:    "644",
	})
	b.Download(config.ConfigDownload{
		Type:    "authority",
		Name:    "clustertls",
		Path:    "/usr/local/share/ca-certificates/extra/cluster.tls.crt",
		Refresh: "24h",
		Mode:    "644",
	})
	b.Download(config.ConfigDownload{
		Type:    "authority",
		Name:    "ssh-user",
		Path:    "/etc/ssh/ssh_user_ca.pub",
		Refresh: "168h", // allow a week for mistakes to be noticed on this one
		Mode:    "644",
	})
	b.Download(config.ConfigDownload{
		Type:    "static",
		Name:    "cluster.conf",
		Path:    "/etc/homeworld/config/cluster.conf",
		Refresh: "24h",
		Mode:    "644",
	})
	b.Download(config.ConfigDownload{
		Type:    "api",
		Name:    "get-local-config",
		Path:    "/etc/homeworld/config/local.conf",
		Refresh: "24h",
		Mode:    "644",
	})
	if variant == MASTER {
		b.Download(config.ConfigDownload{
			Type:    "authority",
			Name:    "serviceaccount",
			Path:    "/etc/homeworld/keys/serviceaccount.pem",
			Refresh: "24h",
			Mode:    "644",
		})
		b.Download(config.ConfigDownload{
			Type:    "api",
			Name:    "fetch-serviceaccount-key",
			Path:    "/etc/homeworld/keys/serviceaccount.key",
			Refresh: "24h",
			Mode:    "600",
		})
		b.Download(config.ConfigDownload{
			Type:    "authority",
			Name:    "etcd-client",
			Path:    "/etc/homeworld/authorities/etcd-client.pem",
			Refresh: "24h",
			Mode:    "644",
		})
		b.Download(config.ConfigDownload{
			Type:    "authority",
			Name:    "etcd-server",
			Path:    "/etc/homeworld/authorities/etcd-server.pem",
			Refresh: "24h",
			Mode:    "644",
		})
	}
}

func (b *ActionBuilder) DefaultKeys(variant string) {
	b.Key(config.ConfigKey{
		Name:      "keygranting",
		Type:      "tls",
		Key:       "/etc/homeworld/keyclient/granting.key",
		Cert:      "/etc/homeworld/keyclient/granting.pem",
		API:       "renew-keygrant",
		InAdvance: "336h", // renew two weeks before expiration
	})
	b.Key(config.ConfigKey{
		Name:      "ssh-host",
		Type:      "ssh-pubkey",
		Key:       "/etc/ssh/ssh_host_rsa_key.pub",
		Cert:      "/etc/ssh/ssh_host_rsa_cert",
		API:       "grant-ssh-host",
		InAdvance: "168h", // renew one week before expiration
	})
	b.Key(config.ConfigKey{
		// for master nodes, worker nodes (both for kubelet), and supervisor nodes (for prometheus)
		Name:      "kube-worker",
		Type:      "tls",
		Key:       "/etc/homeworld/keys/kubernetes-worker.key",
		Cert:      "/etc/homeworld/keys/kubernetes-worker.pem",
		API:       "grant-kubernetes-worker",
		InAdvance: "168h", // renew one week before expiration
	})
	if variant == SUPERVISOR {
		b.Key(config.ConfigKey{
			Name:      "clustertls",
			Type:      "tls",
			Key:       "/etc/homeworld/ssl/homeworld.private.key",
			Cert:      "/etc/homeworld/ssl/homeworld.private.pem",
			API:       "grant-registry-host",
			InAdvance: "168h", // renew one week before expiration
		})
	} else if variant == MASTER {
		b.Key(config.ConfigKey{
			Name:      "kube-master",
			Type:      "tls",
			Key:       "/etc/homeworld/keys/kubernetes-master.key",
			Cert:      "/etc/homeworld/keys/kubernetes-master.pem",
			API:       "grant-kubernetes-master",
			InAdvance: "168h", // renew one week before expiration
		})
		b.Key(config.ConfigKey{
			Name:      "etcd-server",
			Type:      "tls",
			Key:       "/etc/homeworld/keys/etcd-server.key",
			Cert:      "/etc/homeworld/keys/etcd-server.pem",
			API:       "grant-etcd-server",
			InAdvance: "168h", // renew one week before expiration
		})
		b.Key(config.ConfigKey{
			Name:      "etcd-client",
			Type:      "tls",
			Key:       "/etc/homeworld/keys/etcd-client.key",
			Cert:      "/etc/homeworld/keys/etcd-client.pem",
			API:       "grant-etcd-client",
			InAdvance: "168h", // renew one week before expiration
		})
	}
}

type ActionBuilder struct {
	State   *state.ClientState
	Actions []actloop.Action
	Err     error
}

func (b *ActionBuilder) Add(action actloop.Action) {
	if action != nil {
		b.Actions = append(b.Actions, action)
	}
}

func (b *ActionBuilder) AddOrError(action actloop.Action, err error) {
	if err != nil {
		b.Err = multierror.Append(b.Err, err)
	} else {
		b.Add(action)
	}
}

func (b *ActionBuilder) Bootstrap() {
	act, err := keygen.PrepareKeygenAction(config.ConfigKey{Type: "tls", Key: paths.GrantingKeyPath})
	b.AddOrError(act, err)
	act, err = bootstrap.PrepareBootstrapAction(b.State, paths.BootstrapTokenPath, paths.BootstrapTokenAPI)
	b.AddOrError(act, err)
}

func (b *ActionBuilder) Key(key config.ConfigKey) {
	// for generating private keys
	act, err := keygen.PrepareKeygenAction(key)
	b.AddOrError(act, err)
	// for getting certificates for keys
	act, err = keyreq.PrepareRequestOrRenewKeys(b.State, key)
	b.AddOrError(act, err)
}

func (b *ActionBuilder) Download(dl config.ConfigDownload) {
	// for downloading files and public keys
	act, err := download.PrepareDownloadAction(b.State, dl)
	b.AddOrError(act, err)
}

func BuildActions(s *state.ClientState, variant string) ([]actloop.Action, error) {
	b := &ActionBuilder{
		State: s,
	}
	b.Bootstrap()
	b.DefaultKeys(variant)
	b.DefaultDownloads(variant)
	if b.Err != nil {
		return nil, b.Err
	}
	return b.Actions, nil
}
