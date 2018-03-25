#!/bin/bash
set -e -u

# this script runs in each pod in the ceph manager daemonset

if [ "${NODE_HOSTNAME:-}" = "" ]
then
	echo "NO NODE HOSTNAME" 1>&2
	exit 1
fi

ceph-mgr --version  # to help with debugging

if [ ! -e /etc/ceph-keyrings/client.admin.keyring ]
then
	echo "NO ADMIN KEYRING" 1>&2
	exit 1
fi

mkdir -p "/var/lib/ceph/mgr/ceph-${NODE_HOSTNAME}"

# TODO: don't have this container using the client admin keyring for this...
ceph -k /etc/ceph-keyrings/client.admin.keyring auth get-or-create "mgr.${NODE_HOSTNAME}" mon 'allow profile mgr' osd 'allow *' mds 'allow *' -o "/var/lib/ceph/mgr/ceph-${NODE_HOSTNAME}/keyring"

ceph-mgr -d -i "${NODE_HOSTNAME}"
