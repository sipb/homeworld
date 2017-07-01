#!/bin/bash
set -e -u

source /etc/hyades/cluster.conf
source /etc/hyades/local.conf

# allow verification of etcd certs
FLANOPT="--etcd-cafile /etc/hyades/certs/kube/etcd-ca.pem"
# authenticate to etcd servers
FLANOPT="$FLANOPT --etcd-certfile /etc/hyades/certs/kube/etcd-cert.pem --etcd-keyfile /etc/hyades/certs/kube/local-key.pem"
# endpoints
FLANOPT="$FLANOPT --etcd-endpoints ${ETCD_ENDPOINTS}"

FLANOPT="$FLANOPT --iface ${HOST_IP}" # the IP
FLANOPT="$FLANOPT --ip-masq"
FLANOPT="$FLANOPT --public-ip ${HOST_IP}" # the IP

exec /usr/bin/flanneld $FLANOPT
