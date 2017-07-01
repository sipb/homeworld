#!/bin/bash
set -e -u

# use known apiserver
SRVOPT="--kubeconfig=/etc/hyades/kubeconfig"

SRVOPT="$SRVOPT --leader-elect"

exec /usr/bin/hyperkube kube-scheduler $SRVOPT
