package worldconfig

import (
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/bootstrap"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/download"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keyreq"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

const OneDay = 24 * time.Hour
const OneWeek = 7 * OneDay

func (b *ActionBuilder) DefaultDownloads() {
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
	b.FromAPI(
		"fetch-serviceaccount-key",
		"/etc/homeworld/keys/serviceaccount.key",
		OneDay,
		0600,
	)
}

func (b *ActionBuilder) DefaultKeys() {
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
	b.TLSKey(
		"/etc/homeworld/ssl/homeworld.private.key",
		"/etc/homeworld/ssl/homeworld.private.pem",
		"grant-registry-host",
		OneWeek, // renew one week before expiration
	)
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

type ActionBuilder struct {
	State   *state.ClientState
	Context *actloop.NewActionContext
}

func (b *ActionBuilder) Bootstrap() {
	keygen.GenerateKey(paths.GrantingKeyPath, b.Context)
	bootstrap.Bootstrap(paths.BootstrapTokenAPI, b.Context)
}

func (b *ActionBuilder) TLSKey(key string, cert string, api string, inadvance time.Duration) {
	keygen.GenerateKey(key, b.Context)
	keyreq.RequestOrRenewTLSKey(key, cert, api, inadvance, b.Context)
}

func (b *ActionBuilder) SSHCertificate(key string, cert string, api string, inadvance time.Duration) {
	keyreq.RequestOrRenewSSHKey(key, cert, api, inadvance, b.Context)
}

func (b *ActionBuilder) PublicKey(name string, path string, refreshPeriod time.Duration) {
	download.DownloadAuthority(name, path, refreshPeriod, b.Context)
}

func (b *ActionBuilder) StaticFile(name string, path string, refreshPeriod time.Duration) {
	download.DownloadStatic(name, path, refreshPeriod, b.Context)
}

func (b *ActionBuilder) FromAPI(api string, path string, refreshPeriod time.Duration, mode uint64) {
	download.DownloadFromAPI(api, path, refreshPeriod, mode, b.Context)
}

func BuildActions(s *state.ClientState) actloop.NewAction {
	return func(nac *actloop.NewActionContext) {
		b := &ActionBuilder{
			State:   s,
			Context: nac,
		}
		b.Bootstrap()
		b.DefaultKeys()
		b.DefaultDownloads()
	}
}
