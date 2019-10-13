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
		RenewKeygrantAPI,
		nac,
	)
	download.DownloadAuthority(
		KubernetesAuthority,
		paths.KubernetesCAPath,
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		ClusterCAAuthority,
		"/usr/local/share/ca-certificates/extra/cluster.tls.crt",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		SSHUserAuthority,
		"/etc/ssh/ssh_user_ca.pub",
		OneWeek, // allow a week for mistakes to be noticed on this one
		nac,
	)
	download.DownloadStatic(
		ClusterConfStatic,
		paths.ClusterConfPath,
		OneDay,
		nac,
	)
	download.DownloadFromAPI(
		LocalConfAPI,
		paths.LocalConfPath,
		OneDay,
		0644,
		nac,
	)
	hostname.ReloadHostnameFrom(
		paths.LocalConfPath,
		nac,
	)
	download.DownloadAuthority(
		ServiceAccountAuthority,
		"/etc/homeworld/keys/serviceaccount.pem",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		EtcdClientAuthority,
		"/etc/homeworld/authorities/etcd-client.pem",
		OneDay,
		nac,
	)
	download.DownloadAuthority(
		EtcdServerAuthority,
		"/etc/homeworld/authorities/etcd-server.pem",
		OneDay,
		nac,
	)
	download.DownloadFromAPI(
		FetchServiceAccountKeyAPI,
		"/etc/homeworld/keys/serviceaccount.key",
		OneDay,
		0600,
		nac,
	)
	TLSKey(
		"/etc/homeworld/keyclient/granting.key",
		"/etc/homeworld/keyclient/granting.pem",
		RenewKeygrantAPI,
		2*OneWeek, // renew two weeks before expiration
		nac,
	)
	keyreq.RequestOrRenewSSHKey(
		"/etc/ssh/ssh_host_rsa_key.pub",
		"/etc/ssh/ssh_host_rsa_cert",
		SignSSHHostKeyAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		// for master nodes, worker nodes (both for kubelet), and supervisor nodes (for prometheus)
		paths.KubernetesWorkerKey,
		paths.KubernetesWorkerCert,
		SignKubernetesWorkerAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/ssl/homeworld.private.key",
		"/etc/homeworld/ssl/homeworld.private.pem",
		SignRegistryHostAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		paths.KubernetesMasterKey,
		paths.KubernetesMasterCert,
		SignKubernetesMasterAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/keys/etcd-server.key",
		"/etc/homeworld/keys/etcd-server.pem",
		SignEtcdServerAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
	TLSKey(
		"/etc/homeworld/keys/etcd-client.key",
		"/etc/homeworld/keys/etcd-client.pem",
		SignEtcdClientAPI,
		OneWeek, // renew one week before expiration
		nac,
	)
}

func TLSKey(key string, cert string, api string, inadvance time.Duration, nac *actloop.NewActionContext) {
	keygen.GenerateKey(key, nac)
	keyreq.RequestOrRenewTLSKey(key, cert, api, inadvance, nac)
}
