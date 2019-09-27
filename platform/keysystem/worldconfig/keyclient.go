package worldconfig

import (
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/bootstrap"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/download"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/hostname"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keyreq"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
)

const OneDay = 24 * time.Hour
const OneWeek = 7 * OneDay

func ConvergeState(nac *actloop.NewActionContext) {
	keygen.GenerateKey(
		paths.GrantingKeyPath,
		nac,
	)
	bootstrap.Bootstrap(
		paths.BootstrapTokenAPI,
		nac,
	)
	download.DownloadAuthority(
		"kubernetes",
		"/etc/homeworld/authorities/kubernetes.pem",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		"clusterca",
		"/usr/local/share/ca-certificates/extra/cluster.tls.crt",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		"ssh-user",
		"/etc/ssh/ssh_user_ca.pub",
		OneWeek, // allow a week for mistakes to be noticed on this one
		nac,
	)
	download.DownloadStatic(
		"cluster.conf",
		"/etc/homeworld/config/cluster.conf",
		OneDay,
		nac,
	)
	download.DownloadFromAPI(
		"get-local-config",
		"/etc/homeworld/config/local.conf",
		OneDay,
		0644,
		nac,
	)
	hostname.ReloadHostnameFrom(
		"/etc/homeworld/config/local.conf",
		nac,
	)
	download.DownloadAuthority(
		"serviceaccount",
		"/etc/homeworld/keys/serviceaccount.pem",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		"etcd-client",
		"/etc/homeworld/authorities/etcd-client.pem",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		"etcd-server",
		"/etc/homeworld/authorities/etcd-server.pem",
		OneDay,
		nac,
	)
	download.DownloadFromAPI(
		"fetch-serviceaccount-key",
		"/etc/homeworld/keys/serviceaccount.key",
		OneDay,
		0600,
		nac,
	)
	TLSKey(
		"/etc/homeworld/keyclient/granting.key",
		"/etc/homeworld/keyclient/granting.pem",
		"renew-keygrant",
		2*OneWeek, // renew two weeks before expiration
		nac,
	)
	keyreq.RequestOrRenewSSHKey(
		"/etc/ssh/ssh_host_rsa_key.pub",
		"/etc/ssh/ssh_host_rsa_cert",
		"grant-ssh-host",
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		// for master nodes, worker nodes (both for kubelet), and supervisor nodes (for prometheus)
		"/etc/homeworld/keys/kubernetes-worker.key",
		"/etc/homeworld/keys/kubernetes-worker.pem",
		"grant-kubernetes-worker",
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/ssl/homeworld.private.key",
		"/etc/homeworld/ssl/homeworld.private.pem",
		"grant-registry-host",
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/keys/kubernetes-master.key",
		"/etc/homeworld/keys/kubernetes-master.pem",
		"grant-kubernetes-master",
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/keys/etcd-server.key",
		"/etc/homeworld/keys/etcd-server.pem",
		"grant-etcd-server",
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/keys/etcd-client.key",
		"/etc/homeworld/keys/etcd-client.pem",
		"grant-etcd-client",
		OneWeek, // renew one week before expiration
		nac,
	)
}

func TLSKey(key string, cert string, api string, inadvance time.Duration, nac *actloop.NewActionContext) {
	keygen.GenerateKey(key, nac)
	keyreq.RequestOrRenewTLSKey(key, cert, api, inadvance, nac)
}
