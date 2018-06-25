#!/bin/bash
set -e -u
KERNEL=4.9.0-6-amd64

# cdrom drivers
MODULES=(scsi_mod libata ata_piix cdrom sr_mod)
# iso9660 driver
MODULES+=(isofs)
# disk drivers
MODULES+=(virtio virtio_ring virtio_pci virtio_blk)
# ext4 drivers
MODULES+=(mbcache fscrypto jbd2 crc16 crc32c-intel ext4)

SRC="$(dirname "$0")"
OUT="$1"
ROOTIN="$2"
mkdir "${OUT}"
(cd "${OUT}" && mkdir proc dev etc sys bin mod mnt cdrom lib)

# TODO: busybox needs to be checked to be statically linked
cp /bin/busybox "${OUT}/bin/busybox"
for x in $(busybox --list); do ln -s busybox "${OUT}/bin/$x"; done
ln -s bin/busybox "${OUT}/init"
cp "${SRC}/rc" "${OUT}/etc/rc"
cp "${SRC}/inittab" "${OUT}/etc/inittab"
echo "#!/bin/sh" >"${OUT}/etc/module-load"
chmod +x "${OUT}/etc/module-load"
for module in "${MODULES[@]}"
do
	FOUND="$(find "${ROOTIN}/lib/modules/${KERNEL}" -name "${module}.ko" | head -n 1)"
	if [ "$FOUND" == "" ]
	then
		echo "failed to find module: ${module}" 1>&2
		exit 1
	fi
	cp "$FOUND" "${OUT}/mod"
	echo "insmod /mod/${module}.ko" >>"${OUT}/etc/module-load"
done

# these will be deleted later, before packing
cp -R "${ROOTIN}/lib/modules/${KERNEL}" "${OUT}/lib/modules/"
cp "${ROOTIN}/boot/vmlinuz-${KERNEL}" "${OUT}/vmlinuz"
