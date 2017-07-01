#!/bin/bash
set -e -u

source /etc/hyades/cluster.conf

AUTHOPT="--ca-file /etc/hyades/certs/kube/etcd-ca.pem --cert-file /etc/hyades/certs/kube/etcd-cert.pem --key-file /etc/hyades/certs/kube/local-key.pem"

export ETCDCTL_API=2

/usr/bin/etcdctl --endpoints ${ETCD_ENDPOINTS} ${AUTHOPT} set /coreos.com/network/config "{ \"network\": \"${CLUSTER_CIDR}\" }"
