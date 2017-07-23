#!/bin/bash
set -e -u

if [ "${1-}" = "" ] || [ "${2-}" = "" ] || [ "${3-}" = "" ]
then
	echo "Usage: $0 <host> <secretdir> <configdir>" 1>&2
	exit 1
fi

HOST="$1"
SECRETDIR="$2"
CONFIGDIR="$3"

if [ ! -e "$CONFIGDIR/cluster.conf" ]
then
	echo "Invalid configuration directory provided!" 1>&2
	exit 1
fi
if [ ! -e "$SECRETDIR/admission.key" ] || [ ! -e "$SECRETDIR/bootstrap_client_ca.pem" ]
then
	echo "Invalid secrets directory provided!" 1>&2
	exit 1
fi

ETCADMIT="/etc/hyades/admission"

ssh "root@${HOST}" "mkdir -p '${ETCADMIT}/config/' && rm -f '${ETCADMIT}/config/'node-*.conf && apt-get update && apt-get upgrade -y && apt-get install homeworld-admitserver"
scp "$CONFIGDIR/cluster.conf" "$CONFIGDIR"/node-*.conf "root@${HOST}:${ETCADMIT}/config/"
scp "$SECRETDIR/admission.key" "$SECRETDIR/admission.pem" "$SECRETDIR/bootstrap_client_ca.pem" "$SECRETDIR/ssh_host_ca" "$SECRETDIR/ssh_user_ca.pub" "root@${HOST}:${ETCADMIT}/"
ssh "root@${HOST}" systemctl restart admitserver

echo "Finished deploying admitserver."
