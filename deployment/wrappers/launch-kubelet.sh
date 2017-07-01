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

KUBEOPT=""
# just use one API server for now -- TODO: BETTER HIGH-AVAILABILITY
KUBEOPT="$KUBEOPT --kubeconfig=/etc/hyades/kubeconfig --require-kubeconfig"
if [ "x${SCHEDULE_WORK}" = "xtrue" ]
then
	# register as schedulable (i.e. for a worker node)
	KUBEOPT="$KUBEOPT --register-schedulable=true"
else
	if [ "x${SCHEDULE_WORK}" = "xfalse" ]
	then
		# don't register as schedulable (i.e. for a master node)
		KUBEOPT="$KUBEOPT --register-schedulable=false"
	else
		echo 'SCHEDULE_WORK not set!'
		exit 1
	fi
fi
# turn off anonymous authentication
KUBEOPT="$KUBEOPT --anonymous-auth=false"
# add kubelet auth certs
KUBEOPT="$KUBEOPT --tls-cert-file=/etc/hyades/certs/kube/kube-cert.pem --tls-private-key-file=/etc/hyades/certs/kube/local-key.pem"
# add client certificate authority
KUBEOPT="$KUBEOPT --client-ca-file=/etc/hyades/certs/kube/kube-ca.pem"
# turn off cloud provider detection
KUBEOPT="$KUBEOPT --cloud-provider="
# use rkt
KUBEOPT="$KUBEOPT --container-runtime rkt"
# pod manifests
KUBEOPT="$KUBEOPT --pod-manifest-path=/etc/hyades/manifests/"
# DNS
KUBEOPT="$KUBEOPT --cluster-dns ${SERVICE_DNS} --cluster-domain ${CLUSTER_DOMAIN}"
# experimental options to better enforce env config
KUBEOPT="$KUBEOPT --experimental-fail-swap-on"

exec /usr/bin/hyperkube kubelet $KUBEOPT
