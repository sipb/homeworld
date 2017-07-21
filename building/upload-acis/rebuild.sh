#!/bin/bash

set -e -u

cd "$(dirname $0)"
mkdir -p staging
cp ../build-acis/containers/homeworld-*.aci -t staging
for x in staging/homeworld-*.aci
do
	if [ ! -e "$x.asc" ]
	then
		gpg --armor --detach-sign --local-user 0x8422464D9EE78588 "$x"
	fi
done
rsync -av --progress ./staging/* /mit/hyades/acis/
