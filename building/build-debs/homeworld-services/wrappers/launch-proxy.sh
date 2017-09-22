#!/bin/bash
set -e -u

# use known apiserver (depends on kubelet generator running)
SRVOPT=(--kubeconfig=/etc/homeworld/config/kubeconfig)
# synchronize every minute (TODO: IS THIS A GOOD AMOUNT OF TIME?)
SRVOPT+=(--config-sync-period=1m)

exec /usr/bin/hyperkube kube-proxy "${SRVOPT[@]}"
