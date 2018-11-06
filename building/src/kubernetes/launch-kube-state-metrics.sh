#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

# TODO: don't duplicate this from launch-kubelet.sh
cat >/etc/homeworld/config/kubeconfig <<EOCONFIG
current-context: hyades
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority: /etc/homeworld/authorities/kubernetes.pem
    server: ${APISERVER}
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

# TODO: only mount the required keys, not everything
TLS_MOUNTPOINT=/etc/homeworld/
TLS_STORAGE=/etc/homeworld/
KSM_IMAGE=homeworld.private/kube-state-metrics:1.2.0-4

# provide directory for kubernetes TLS certificates
HOSTOPT=(--volume "kube-certs,kind=host,readOnly=true,source=${TLS_STORAGE}" --mount "volume=kube-certs,target=${TLS_MOUNTPOINT}")
# bind ports to public interface
HOSTOPT+=(--port=metrics:9104 --port=metametrics:9105)
# don't use KVM, because port forwarding
HOSTOPT+=(--stage1-path /usr/lib/rkt/stage1-images/stage1-coreos.aci)

# provide kubeconfig from kubelet
KSMOPT=(--kubeconfig /etc/homeworld/config/kubeconfig --in-cluster=false)

exec rkt run "${HOSTOPT[@]}" "${KSM_IMAGE}" -- "${KSMOPT[@]}"
