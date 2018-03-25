#!/bin/bash
set -e -u

# this script runs in each pod in the ceph OSD daemonset

# responsibilities:
#     initialize OSD on first run
#         generate UUID for OSD
#         generate key for OSD
#         bootstrap OSD
#         ceph mkfs
#         ceph-osd mkfs
#     mount device
#     launch ceph OSD

META_MOUNT="/etc/ceph-osd-meta/"  # preserved across container restarts
DEVICE="/dev/osd-disk"
OSDDIR="/var/lib/ceph-homeworld/osd/"  # **NOT** preserved across container restarts

ceph-osd --version  # to help with debugging

mkdir -p "${OSDDIR}"

if [ ! -e "${META_MOUNT}/inited" ]
then
        if [ ! -e /etc/ceph-keyrings/client.bootstrap-osd.keyring ]
        then
                echo "NO BOOTSTRAP KEYRING" 1>&2
                exit 1
        fi
        if [ -e /etc/ceph-keyrings/mon.keyring ]
        then
                echo "should not have access to monitor keyring!" 1>&2
                exit 1
        fi

        UUID="$(uuidgen)"
	OSD_SECRET="$(ceph-authtool --gen-print-key)"

	# TODO: set up lockbox

	touch "${META_MOUNT}/inited"   # this makes sure that we don't just keep creating more OSDs over time if creation fails

	# TODO: don't do this on the worker node
	OSD_ID="$(echo '{"cephx_secret": "'"$OSD_SECRET"'"}' | ceph osd new "$UUID" -i - -n client.bootstrap-osd -k /etc/ceph-keyrings/client.bootstrap-osd.keyring)"

	mkdir "${OSDDIR}/ceph-${OSD_ID}"

	ceph mon getmap -n client.bootstrap-osd -k /etc/ceph-keyrings/client.bootstrap-osd.keyring -o "${OSDDIR}/ceph-${OSD_ID}/activate.monmap"
	ceph-authtool "${OSDDIR}/ceph-${OSD_ID}/keyring" --create-keyring --name "osd.${OSD_ID}" --add-key "${OSD_SECRET}"

	ln -snf "${DEVICE}" "${OSDDIR}/ceph-${OSD_ID}/block"
	if [ ! -e "${OSDDIR}/ceph-${OSD_ID}/keyring" ]
	then
		echo "failed to init keyring properly" 1>&2
		exit 1
	fi
	ceph-osd --osd-objectstore bluestore --mkfs -i "${OSD_ID}" --monmap "${OSDDIR}/ceph-${OSD_ID}/activate.monmap" --keyfile "${OSDDIR}/ceph-${OSD_ID}/keyring" --osd-data "${OSDDIR}/ceph-${OSD_ID}" --osd-uuid "${UUID}"

	echo "${OSD_ID}" >"${META_MOUNT}/inited"
	rm -rf "${OSDDIR}/ceph-${OSD_ID}"
else
	OSD_ID="$(cat "${META_MOUNT}/inited")"
	if [ "${OSD_ID}" = "" ]
	then
		echo "no valid OSD_ID found" 1>&2
		exit 1
	fi
fi

mkdir "${OSDDIR}/ceph-${OSD_ID}"
ceph-bluestore-tool prime-osd-dir --dev "${DEVICE}" --path "${OSDDIR}/ceph-${OSD_ID}"

ceph-osd -d -i "${OSD_ID}" --osd-data "${OSDDIR}/ceph-${OSD_ID}"
