#!/bin/bash
set -e -u

source /etc/hyades/cluster.conf
source /etc/hyades/local.conf

# use known apiserver (depends on kubelet config)
SRVOPT=(--kubeconfig=/etc/hyades/kubeconfig)

SRVOPT+=(--cluster-cidr="${CLUSTER_CIDR}")
SRVOPT+=(--node-cidr-mask-size=24)
SRVOPT+=(--service-cluster-ip-range="${SERVICE_CIDR}")
SRVOPT+=(--cluster-name=hyades)

SRVOPT+=(--leader-elect)

# granting service tokens
SRVOPT+=(--service-account-private-key-file=/etc/hyades/certs/kube/serviceaccount.key)
SRVOPT+=(--root-ca-file=/etc/hyades/certs/kube/kube-ca.pem)

exec /usr/bin/hyperkube kube-controller-manager "${SRVOPT[@]}"
