#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

TLS_STORAGE=/etc/homeworld/

# etcd node name
ETCDOPT=(--name="${HOST_NODE}")
# public advertisement URLs
ETCDOPT+=(--advertise-client-urls="https://${HOST_IP}:2379" --initial-advertise-peer-urls="https://${HOST_IP}:2380")
# listening URLs
ETCDOPT+=(--listen-client-urls=https://0.0.0.0:2379 --listen-peer-urls=https://0.0.0.0:2380)
# initial cluster setup
ETCDOPT+=(--initial-cluster="${ETCD_CLUSTER}" --initial-cluster-token="${ETCD_TOKEN}" --initial-cluster-state=new)
# client-to-server TLS certs
ETCDOPT+=(--cert-file="${TLS_STORAGE}/keys/etcd-server.pem" --key-file="${TLS_STORAGE}/keys/etcd-server.key" --client-cert-auth --trusted-ca-file="${TLS_STORAGE}/authorities/etcd-client.pem")
# server-to-server TLS certs
ETCDOPT+=(--peer-cert-file="${TLS_STORAGE}/keys/etcd-server.pem" --peer-key-file="${TLS_STORAGE}/keys/etcd-server.key" --peer-client-cert-auth --peer-trusted-ca-file="${TLS_STORAGE}/authorities/etcd-server.pem")

exec /usr/bin/etcd "${ETCDOPT[@]}"
