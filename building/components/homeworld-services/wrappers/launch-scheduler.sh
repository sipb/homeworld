#!/bin/bash
set -e -u

# use known apiserver
SRVOPT=(--kubeconfig=/etc/homeworld/config/kubeconfig)

SRVOPT+=(--leader-elect)

exec /usr/bin/hyperkube kube-scheduler "${SRVOPT[@]}"
