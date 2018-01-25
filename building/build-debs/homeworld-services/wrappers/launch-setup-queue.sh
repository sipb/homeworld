#!/bin/bash
set -e -u

# TODO: make this not dependent on kube-state-metrics populating kubeconfig

QUEUEDIR="/etc/homeworld/deployqueue/"

if [ ! -e "${QUEUEDIR}" ]
then
	echo "No queue directory."
	exit
fi

while [ "$(ls "${QUEUEDIR}" | wc -l)" -gt 0 ]
do
	NEXT="$(ls "${QUEUEDIR}" | head -n 1)"
	echo "Queuing ${NEXT}..."
	hyperkube kubectl apply --kubeconfig /etc/homeworld/config/kubeconfig -f "${QUEUEDIR}/${NEXT}"
	rm "${QUEUEDIR}/${NEXT}"
done

echo "Done queuing!"
