#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

TLS_STORAGE=/etc/homeworld/

exec /usr/bin/etcd-metrics-exporter "https://${HOST_IP}:2379" "${TLS_STORAGE}/authorities/etcd-server.pem" "${TLS_STORAGE}/keys/etcd-client.key" "${TLS_STORAGE}/keys/etcd-client.pem"
