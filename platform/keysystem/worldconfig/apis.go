package worldconfig

// NOTE: we're trying to centralize all of these names into this file, but that process isn't complete, and there are
// likely to be a number of uses that haven't been updated (or can't be, especially for spire code)

// when these names are changed, the authorities.tgz must be regenerated
const KeygrantingAuthority = "keygranting"
const KubernetesAuthority = "kubernetes"
const ClusterCAAuthority = "clusterca"
const SSHHostAuthority = "ssh-host"
const SSHUserAuthority = "ssh-user"
const ServiceAccountAuthority = "serviceaccount"
const EtcdServerAuthority = "etcd-server"
const EtcdClientAuthority = "etcd-client"

const ClusterConfStatic = "cluster.conf"

const ListAdmitRequestsAPI = "list-admits"
const ApproveAdmitAPI = "approve-admit"
const RenewKeygrantAPI = "renew-keygrant"
const ImpersonateKerberosAPI = "auth-to-kerberos"
const LocalConfAPI = "get-local-config"

const FetchServiceAccountKeyAPI = "fetch-serviceaccount-key"
const SignKubernetesWorkerAPI = "grant-kubernetes-worker"
const SignKubernetesMasterAPI = "grant-kubernetes-master"
const SignSSHHostKeyAPI = "grant-ssh-host"
const SignRegistryHostAPI = "grant-registry-host"
const SignEtcdServerAPI = "grant-etcd-server"
const SignEtcdClientAPI = "grant-etcd-client"

const AccessSSHAPI = "access-ssh"
const AccessEtcdAPI = "access-etcd"
const AccessKubernetesAPI = "access-kubernetes"
