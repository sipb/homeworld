#!/bin/bash
set -e -u

# this script runs in each pod in the ceph monitor daemonset

# responsibilities:
#     initialize monitor on first run
#         collect keyring from kubernetes
#         monmapgen
#         ceph-mon mkfs
#         signal ready to start
#     launch ceph monitor

if [ "${NODE_HOSTNAME:-}" = "" ]
then
	echo "NO NODE HOSTNAME" 1>&2
	exit 1
fi

STORAGE_MOUNT="/var/lib/ceph/mon"

ceph-mon --version  # to help with debugging

MONDIR="${STORAGE_MOUNT}/ceph-${NODE_HOSTNAME}"

if [ ! -e "${MONDIR}" ]
then
	if [ ! -e /etc/ceph-keyrings/mon.keyring ]
	then
		echo "NO MON KEYRING" 1>&2
		exit 1
	fi

	rm -rf "${STORAGE_MOUNT}/ceph-mon-tmp-*"
	WORKDIR="$(mktemp -d --suffix "-${NODE_HOSTNAME}" ceph-mon-tmp-XXXXXXXX -p "${STORAGE_MOUNT}")"
	WORKTMP="$(mktemp -d)"

	monmaptool --create "${WORKTMP}/monmap" --fsid "$(cat /etc/ceph/fs.uuid)" $(cat /etc/ceph/master.list | sed "s/^/--add /")

	ceph-mon --mkfs -d -i "${NODE_HOSTNAME}" --monmap "${WORKTMP}/monmap" --keyring /etc/ceph-keyrings/mon.keyring --mon-data "${WORKDIR}"

	touch "${WORKDIR}/done"

	# atomically rename new configuration to destination
	mv "${WORKDIR}" -T "${MONDIR}"
fi

ceph-mon -d -i "${NODE_HOSTNAME}" --mon-data "${MONDIR}"
