#!/bin/bash
set -e -u

# use known apiserver (depends on kubelet generator running)
SRVOPT="--kubeconfig=/etc/hyades/kubeconfig"
# synchronize every minute (TODO: IS THIS A GOOD AMOUNT OF TIME?)
SRVOPT="$SRVOPT --config-sync-period=1m"

exec /usr/bin/hyperkube kube-proxy $SRVOPT
