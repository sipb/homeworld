#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

# TODO: only mount the required keys, not everything
TLS_MOUNTPOINT=/etc/homeworld/
TLS_STORAGE=/etc/homeworld/
PERSISTENT_DATA=/var/lib/etcd
ETCD_IMAGE=/usr/lib/hyades/images/etcd-current-linux-amd64.aci

mkdir -p ${PERSISTENT_DATA}

# our image is stored on disk -- better hope that's safe!
HOSTOPT=(--insecure-options=image)
# provide data directory for etcd to store persistent data
HOSTOPT+=(--volume "data-dir,kind=host,source=${PERSISTENT_DATA}")
# provide directory for etcd TLS certificates
HOSTOPT+=(--volume "etcd-certs,kind=host,readOnly=true,source=${TLS_STORAGE}" --mount "volume=etcd-certs,target=${TLS_MOUNTPOINT}")
# bind ports to public interface
HOSTOPT+=(--port=client:2379 --port=peer:2380)

# etcd node name
ETCDOPT=(--name="${HOST_NODE}")
# public advertisement URLs
ETCDOPT+=(--advertise-client-urls="https://${HOST_IP}:2379" --initial-advertise-peer-urls="https://${HOST_IP}:2380")
# listening URLs
ETCDOPT+=(--listen-client-urls=https://0.0.0.0:2379 --listen-peer-urls=https://0.0.0.0:2380)
# initial cluster setup
ETCDOPT+=(--initial-cluster="${ETCD_CLUSTER}" --initial-cluster-token="${ETCD_TOKEN}" --initial-cluster-state=new)
# client-to-server TLS certs
ETCDOPT+=(--cert-file="${TLS_MOUNTPOINT}/keys/etcd-server.pem" --key-file="${TLS_MOUNTPOINT}/keys/etcd-server.key" --client-cert-auth --trusted-ca-file="${TLS_MOUNTPOINT}/authorities/etcd-client.pem")
# server-to-server TLS certs
ETCDOPT+=(--peer-cert-file="${TLS_MOUNTPOINT}/keys/etcd-server.pem" --peer-key-file="${TLS_MOUNTPOINT}/keys/etcd-server.key" --peer-client-cert-auth --peer-trusted-ca-file="${TLS_MOUNTPOINT}/authorities/etcd-server.pem")

exec rkt run "${HOSTOPT[@]}" "${ETCD_IMAGE}" -- "${ETCDOPT[@]}"
