#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

# use known apiserver (depends on kubelet config)
SRVOPT=(--kubeconfig=/etc/homeworld/config/kubeconfig)

SRVOPT+=(--cluster-cidr="${CLUSTER_CIDR}")
SRVOPT+=(--node-cidr-mask-size=24)
SRVOPT+=(--service-cluster-ip-range="${SERVICE_CIDR}")
SRVOPT+=(--cluster-name=hyades)

SRVOPT+=(--leader-elect)

SRVOPT+=(--allocate-node-cidrs)

# granting service tokens
SRVOPT+=(--service-account-private-key-file=/etc/homeworld/keys/serviceaccount.key)
SRVOPT+=(--root-ca-file=/etc/homeworld/authorities/kubernetes.pem)

exec /usr/bin/hyperkube kube-controller-manager "${SRVOPT[@]}"
