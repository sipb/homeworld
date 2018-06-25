#!/bin/sh
set -e -u

# TODO: autodetect??
DEST_DEVICE=/dev/vda
HOSTNAME=egg-sandwich
DNS_SERVERS="18.70.0.160 18.71.0.151 18.72.0.3"
INTERFACE=ens3
ADDRESS=18.4.60.150/23
GATEWAY=18.4.60.1
ROOT=password

PART=1
DEST_PART="${DEST_DEVICE}${PART}"

dd if=/dev/zero of="$DEST_DEVICE" count=1024
echo "n p 1   w" | tr " " "\n" | fdisk "$DEST_DEVICE"
echo "writing image..."
gunzip -c disk.img.gz | dd of="$DEST_PART"

mkdir -p /mnt
mount -t ext4 "$DEST_PART" /mnt

echo "${DEST_PART} / ext4 errors=remount-ro 0 1" >>/mnt/etc/fstab
echo "$HOSTNAME" >/mnt/etc/hostname

for server in ${DNS_SERVERS}
do
	echo "nameserver $server"
done >/mnt/etc/resolv.conf

cat >>/mnt/etc/network/interfaces <<EOF
auto ${INTERFACE}
iface ${INTERFACE} inet static
	address ${ADDRESS}
	gateway ${GATEWAY}
EOF


mount --bind /dev /mnt/dev
mount -t proc proc /mnt/proc
mount -t sysfs none /mnt/sys
chroot /mnt /bin/bash -c "echo 'root:${ROOT}' | chpasswd && resize2fs '$DEST_PART' && grub-install '$DEST_DEVICE' && update-grub"
umount /mnt/sys
umount /mnt/proc
umount /mnt/dev
umount /mnt

reboot
