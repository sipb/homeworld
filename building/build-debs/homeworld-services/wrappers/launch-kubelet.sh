#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

TMP_KUBECONFIG="/etc/homeworld/config/kubeconfig-tmp.$$"
cat >"$TMP_KUBECONFIG" <<EOCONFIG
current-context: hyades
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority: /etc/homeworld/authorities/kubernetes.pem
    servers:
EOCONFIG

for apiserver in ${APISERVERS}
do
    echo "    - ${apiserver}" >>"$TMP_KUBECONFIG"
done

cat >>"$TMP_KUBECONFIG" <<EOCONFIG
  name: hyades-cluster
users:
- name: kubelet-auth
  user:
    client-certificate: /etc/homeworld/keys/kubernetes-worker.pem
    client-key: /etc/homeworld/keys/kubernetes-worker.key
contexts:
- context:
    cluster: hyades-cluster
    user: kubelet-auth
  name: hyades
EOCONFIG

mv "$TMP_KUBECONFIG" /etc/homeworld/config/kubeconfig

KUBEOPT=()
# just use one API server for now -- TODO: BETTER HIGH-AVAILABILITY
KUBEOPT+=(--kubeconfig=/etc/homeworld/config/kubeconfig --require-kubeconfig)
if [ "${SCHEDULE_WORK}" = "true" ]
then
	# register as schedulable (i.e. for a worker node)
	KUBEOPT+=(--register-schedulable=true)
else
	if [ "${SCHEDULE_WORK}" = "false" ]
	then
		# don't register as schedulable (i.e. for a master node)
		KUBEOPT+=(--register-schedulable=false)
	else
		echo 'SCHEDULE_WORK not set!'
		exit 1
	fi
fi
KUBEOPT+=(--allow-privileged=true)
# turn off anonymous authentication
KUBEOPT+=(--anonymous-auth=false)
# add kubelet auth certs
KUBEOPT+=(--tls-cert-file=/etc/homeworld/keys/kubernetes-worker.pem --tls-private-key-file=/etc/homeworld/keys/kubernetes-worker.key)
# add client certificate authority
KUBEOPT+=(--client-ca-file=/etc/homeworld/authorities/kubernetes.pem)
# turn off cloud provider detection
KUBEOPT+=(--cloud-provider=)
# use rkt
KUBEOPT+=(--container-runtime rkt)
# pod manifests
KUBEOPT+=(--pod-manifest-path=/etc/hyades/manifests/)
# DNS
KUBEOPT+=(--cluster-dns "${SERVICE_DNS}" --cluster-domain "${CLUSTER_DOMAIN}")

exec /usr/bin/hyperkube kubelet "${KUBEOPT[@]}"
