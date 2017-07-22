#!/bin/bash
set -e -u

if [ "${1-}" = "" ]
then
	echo "Usage: $0 <...>/cluster.conf" 1>&2
	exit 1
fi

rm -f flannel.yml

ADDRESS="$(grep 'CLUSTER_CIDR=' "${1}" | sed 's/^\(.*\)=\(.*\)$/\2/g')"
if [ "$(echo "$ADDRESS" | tr "./" "\n\n" | wc -l)" != 5 ]
then
	echo "Invalid address."
	exit 1
fi

sed "s|{{NETWORK}}|${ADDRESS}|g" >flannel.yml <flannel.yml.in
