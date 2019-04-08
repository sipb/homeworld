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

exec /usr/bin/kube-state-metrics --kubeconfig /etc/homeworld/config/kubeconfig --port 9104 --telemetry-port 9105
