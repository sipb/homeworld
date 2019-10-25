#!/bin/sh
set -e -u
echo "launching prepartition"

# get the /dev/sdX device for the installer, stripping off the partition number.
ISOIMAGE="$(readlink -f /dev/disk/by-label/ISOIMAGE | tr -d 1234567890)"

CHOSEN_DISK=""

# assuming that there are two disks, the installer and the hard disk, select the hard disk
for disk in $(list-devices disk)
do
    if [ "${disk}" != "${ISOIMAGE}" ]
    then
        if [ "${CHOSEN_DISK}" = "" ]
        then
            CHOSEN_DISK="${disk}"
        else
            echo "too many valid disks: at least ${CHOSEN_DISK} ${disk}"
            exit 1
        fi
    fi
done

if [ "${CHOSEN_DISK}" = "" ]
then
    echo "no available disks"
    exit 1
fi

echo "selected device: ${CHOSEN_DISK}"
debconf-set partman-auto/disk "${CHOSEN_DISK}"
debconf-set grub-installer/bootdev "${CHOSEN_DISK}"
