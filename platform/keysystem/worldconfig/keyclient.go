package worldconfig

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sipb/homeworld/platform/util/certutil"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/bootstrap"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/download"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keyreq"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
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

const OneDay = 24 * time.Hour
const OneWeek = 7 * OneDay

func (b *ActionBuilder) DefaultDownloads(variant string) {
	b.PublicKey(
		"kubernetes",
		"/etc/homeworld/authorities/kubernetes.pem",
		OneDay,
	)
	b.PublicKey(
		"clustertls",
		"/usr/local/share/ca-certificates/extra/cluster.tls.crt",
		OneDay,
	)
	b.PublicKey(
		"ssh-user",
		"/etc/ssh/ssh_user_ca.pub",
		OneWeek, // allow a week for mistakes to be noticed on this one
	)
	b.StaticFile(
		"cluster.conf",
		"/etc/homeworld/config/cluster.conf",
		OneDay,
	)
	b.FromAPI(
		"get-local-config",
		"/etc/homeworld/config/local.conf",
		OneDay,
		0644,
	)
	b.PublicKey(
		"serviceaccount",
		"/etc/homeworld/keys/serviceaccount.pem",
		OneDay,
	)
	b.PublicKey(
		"etcd-client",
		"/etc/homeworld/authorities/etcd-client.pem",
		OneDay,
	)
	b.PublicKey(
		"etcd-server",
		"/etc/homeworld/authorities/etcd-server.pem",
		OneDay,
	)
	if variant == MASTER {
		b.FromAPI(
			"fetch-serviceaccount-key",
			"/etc/homeworld/keys/serviceaccount.key",
			OneDay,
			0600,
		)
	}
}

func (b *ActionBuilder) DefaultKeys(variant string) {
	b.TLSKey(
		"/etc/homeworld/keyclient/granting.key",
		"/etc/homeworld/keyclient/granting.pem",
		"renew-keygrant",
		2*OneWeek, // renew two weeks before expiration
	)
	b.SSHCertificate(
		"/etc/ssh/ssh_host_rsa_key.pub",
		"/etc/ssh/ssh_host_rsa_cert",
		"grant-ssh-host",
		OneWeek, // renew one week before expiration
	)
	b.TLSKey(
		// for master nodes, worker nodes (both for kubelet), and supervisor nodes (for prometheus)
		"/etc/homeworld/keys/kubernetes-worker.key",
		"/etc/homeworld/keys/kubernetes-worker.pem",
		"grant-kubernetes-worker",
		OneWeek, // renew one week before expiration
	)
	if variant == SUPERVISOR {
		b.TLSKey(
			"/etc/homeworld/ssl/homeworld.private.key",
			"/etc/homeworld/ssl/homeworld.private.pem",
			"grant-registry-host",
			OneWeek, // renew one week before expiration
		)
	} else if variant == MASTER {
		b.TLSKey(
			"/etc/homeworld/keys/kubernetes-master.key",
			"/etc/homeworld/keys/kubernetes-master.pem",
			"grant-kubernetes-master",
			OneWeek, // renew one week before expiration
		)
		b.TLSKey(
			"/etc/homeworld/keys/etcd-server.key",
			"/etc/homeworld/keys/etcd-server.pem",
			"grant-etcd-server",
			OneWeek, // renew one week before expiration
		)
		b.TLSKey(
			"/etc/homeworld/keys/etcd-client.key",
			"/etc/homeworld/keys/etcd-client.pem",
			"grant-etcd-client",
			OneWeek, // renew one week before expiration
		)
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

func (b *ActionBuilder) Error(err error) {
	b.Err = multierror.Append(b.Err, err)
}

func (b *ActionBuilder) AddOrError(action actloop.Action, err error) {
	if err != nil {
		b.Error(err)
	} else {
		b.Add(action)
	}
}

func (b *ActionBuilder) Bootstrap() {
	b.Add(keygen.TLSKeygenAction{Keypath: paths.GrantingKeyPath, Bits: keygen.DefaultRSAKeyLength})

	b.Add(&bootstrap.BootstrapAction{
		State:         b.State,
		TokenFilePath: paths.BootstrapTokenPath,
		TokenAPI:      paths.BootstrapTokenAPI,
	})
}

func (b *ActionBuilder) TLSKey(key string, cert string, api string, inadvance time.Duration) {
	b.Add(keygen.TLSKeygenAction{Keypath: key, Bits: keygen.DefaultRSAKeyLength})
	// for generating private keys
	b.Add(&keyreq.RequestOrRenewAction{
		State:           b.State,
		InAdvance:       inadvance,
		API:             api,
		KeyFile:         key,
		CertFile:        cert,
		CheckExpiration: certutil.CheckTLSCertExpiration,
		GenCSR:          csrutil.BuildTLSCSR,
	})
}

func (b *ActionBuilder) SSHCertificate(key string, cert string, api string, inadvance time.Duration) {
	// for getting certificates for keys
	b.Add(&keyreq.RequestOrRenewAction{
		State:           b.State,
		InAdvance:       inadvance,
		API:             api,
		KeyFile:         key,
		CertFile:        cert,
		CheckExpiration: certutil.CheckSSHCertExpiration,
		GenCSR:          csrutil.BuildSSHCSR,
	})
}

func (b *ActionBuilder) PublicKey(name string, path string, refreshPeriod time.Duration) {
	b.Add(&download.DownloadAction{Fetcher: &download.AuthorityFetcher{Keyserver: b.State.Keyserver, AuthorityName: name}, Path: path, Refresh: refreshPeriod, Mode: 0644})
}

func (b *ActionBuilder) StaticFile(name string, path string, refreshPeriod time.Duration) {
	b.Add(&download.DownloadAction{Fetcher: &download.StaticFetcher{Keyserver: b.State.Keyserver, StaticName: name}, Path: path, Refresh: refreshPeriod, Mode: 0644})
}

func (b *ActionBuilder) FromAPI(name string, path string, refreshPeriod time.Duration, mode uint64) {
	b.Add(&download.DownloadAction{Fetcher: &download.APIFetcher{State: b.State, API: name}, Path: path, Refresh: refreshPeriod, Mode: mode})
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
