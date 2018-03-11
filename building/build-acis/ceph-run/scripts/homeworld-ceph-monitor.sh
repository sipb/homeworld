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

monmaptool --create /tmp/monmap --fsid "$(cat /etc/ceph/fs.uuid)" $(cat /etc/ceph/master.list | sed "s/^/--add /")

TODO: figure out how to get this hostname
mkdir /var/lib/ceph/mon/ceph-{hostname}

false
