#!/bin/sh
set -e -u

# TODO: autodetect some settings??
DEST_DEVICE=/dev/vda
INTERFACE=ens3
source /cdrom/settings

PART=1
DEST_PART="${DEST_DEVICE}${PART}"

dd if=/dev/zero of="$DEST_DEVICE" count=1024
echo "n p 1   w" | tr " " "\n" | fdisk "$DEST_DEVICE"
echo "writing image..."
gunzip -c disk.img.gz | dd of="$DEST_PART"

mkdir -p /mnt
mount -t ext4 "$DEST_PART" /mnt

echo "${DEST_PART} / ext4 errors=remount-ro 0 1" >>/mnt/etc/fstab

for server in ${DNS_SERVERS}
do
    echo "nameserver $server"
done >/mnt/etc/resolv.conf

mkdir -p /mnt/etc/homeworld/keyclient /mnt/etc/homeworld/config
cp /cdrom/keyservertls.pem /mnt/etc/homeworld/keyclient/keyservertls.pem
cp /cdrom/keyclient-*.yaml /mnt/etc/homeworld/config/
cp /cdrom/keyclient-base.yaml /mnt/etc/homeworld/config/keyclient.yaml
cat /cdrom/dns_bootstrap_lines >>/mnt/etc/hosts

mount --bind /dev /mnt/dev
mount -t proc proc /mnt/proc
mount -t sysfs none /mnt/sys
echo "ISO used to install this node generated at: ${BUILDDATE}" >>/mnt/etc/issue
echo "Git commit used to build the version: ${GIT_HASH}" >>/mnt/etc/issue
echo "SSH host key fingerprints: (as of install)" >>/mnt/etc/issue
chroot /mnt /bin/bash -c "(echo 'root:${PASSWORD}' | chpasswd) && resize2fs '$DEST_PART' && (for x in /etc/ssh/ssh_host_*.pub; do ssh-keygen -l -f \$x; ssh-keygen -l -E md5 -f \$x; done) >>/etc/issue"
echo >>/mnt/etc/issue

read -p 'hostname> ' HOSTNAME
echo "address format: ${ADDRESS_PREFIX}<address infix>${ADDRESS_SUFFIX}"
read -p 'address infix> ' ADDRESS
read -p 'bootstrap-token> ' BTOKEN
while [ "$BTOKEN" != "manual" ] && ! chroot /mnt /usr/local/bin/check-token.sh "$BTOKEN"; do
    echo "token did not pass checksum test"
    read -p 'bootstrap-token> ' BTOKEN
done

echo "$HOSTNAME" >/mnt/etc/hostname
echo "${ADDRESS_PREFIX}${ADDRESS}${ADDRESS_SUFFIX} ${HOSTNAME}.${DOMAIN}" >>/mnt/etc/hosts

cat >>/mnt/etc/network/interfaces <<EOF
auto ${INTERFACE}
iface ${INTERFACE} inet static
	address ${ADDRESS_PREFIX}${ADDRESS}${ADDRESS_SUFFIX}
	gateway ${GATEWAY}
EOF

if [ "$BTOKEN" = "manual" ]
then
    mkdir -p /mnt/root/.ssh/
    cp /cdrom/authorized.pub /mnt/root/.ssh/authorized_keys
else
    echo "$BTOKEN" > /mnt/etc/homeworld/keyclient/bootstrap.token
fi

chroot /mnt /bin/bash -c "grub-install '$DEST_DEVICE' && update-grub"
umount /mnt/sys
umount /mnt/proc
umount /mnt/dev
umount /mnt

reboot
