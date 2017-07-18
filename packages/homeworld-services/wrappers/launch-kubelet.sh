#!/bin/bash
set -e -u

source /etc/hyades/cluster.conf
source /etc/hyades/local.conf

cat >/etc/hyades/kubeconfig <<EOCONFIG
current-context: hyades
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority: /etc/hyades/certs/kube/kube-ca.pem
    server: ${APISERVER}
  name: hyades-cluster
users:
- name: kubelet-auth
  user:
    client-certificate: /etc/hyades/certs/kube/kube-cert.pem
    client-key: /etc/hyades/certs/kube/local-key.pem
contexts:
- context:
    cluster: hyades-cluster
    user: kubelet-auth
  name: hyades
EOCONFIG

KUBEOPT=()
# just use one API server for now -- TODO: BETTER HIGH-AVAILABILITY
KUBEOPT+=(--kubeconfig=/etc/hyades/kubeconfig --require-kubeconfig)
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
KUBEOPT+=(--tls-cert-file=/etc/hyades/certs/kube/kube-cert.pem --tls-private-key-file=/etc/hyades/certs/kube/local-key.pem)
# add client certificate authority
KUBEOPT+=(--client-ca-file=/etc/hyades/certs/kube/kube-ca.pem)
# turn off cloud provider detection
KUBEOPT+=(--cloud-provider=)
# use rkt
KUBEOPT+=(--container-runtime rkt)
# pod manifests
KUBEOPT+=(--pod-manifest-path=/etc/hyades/manifests/)
# DNS
KUBEOPT+=(--cluster-dns "${SERVICE_DNS}" --cluster-domain "${CLUSTER_DOMAIN}")
# experimental options to better enforce env config
KUBEOPT+=(--experimental-fail-swap-on)

exec /usr/bin/hyperkube kubelet "${KUBEOPT[@]}"
