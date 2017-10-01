#!/bin/bash
if [ "${1:-}" = "" ] || [ "${2:-}" = "" ] || [ "${3:-}" = "" ]
then
	echo "Usage: $0 <confgen-folder> <keyserver-pubkey> <authorized-key>" 1>&2
	echo "Note: authorized-key is only used when selected during install"
	exit 1
fi
set -e -u
CONFGEN_FOLDER=$1
KEYSERVER_PUBKEY=$2
AUTHORIZED_KEY=$3

if [ -e loopdir ]
then
	sudo umount loopdir || true
	rmdir loopdir
else
	mkdir loopdir
fi

rm -rf cd
mkdir cd
sudo mount -o loop debian-9.0.0-amd64-mini.iso loopdir
rsync -q -a -H --exclude=TRANS.TBL loopdir/ cd
sudo umount loopdir
rmdir loopdir
chmod +w --recursive cd
gunzip cd/initrd.gz

PACKAGES="homeworld-apt-setup homeworld-knc homeworld-keysystem"

PASS=$(pwgen 20 1)
echo "Generated password: $PASS"
sed "s|{{HASH}}|$(echo "${PASS}" | mkpasswd -s -m sha-512)|" preseed.cfg.in >preseed.cfg

PACKAGES_VERSIONED=""
for x in $PACKAGES
do
    PACKAGE_VERSIONED="$(./utils/hpm.py ${x})"
    PACKAGES_VERSIONED="${PACKAGES_VERSIONED}"$'\n'"${PACKAGE_VERSIONED}"
    cp "../build-debs/binaries/${PACKAGE_VERSIONED}" .
done

cp "${KEYSERVER_PUBKEY}" keyservertls.pem
cp "${CONFGEN_FOLDER}/"keyclient-{base,supervisor,worker,master}.yaml .
cp "${AUTHORIZED_KEY}" authorized.pub
cat sshd_config.for_hyades >sshd_config.new  # it's a symbolic link...

echo "Packages: ${PACKAGES_VERSIONED}"

cpio -o -H newc -A -F cd/initrd <<EOF
${PACKAGES_VERSIONED}
authorized.pub
keyservertls.pem
postinstall.sh
keyclient-base.yaml
keyclient-supervisor.yaml
keyclient-worker.yaml
keyclient-master.yaml
preseed.cfg
sshd_config.new
EOF

gzip cd/initrd
(cd cd && find . -follow -type f -print0 | xargs -0 md5sum > md5sum.txt)
genisoimage -quiet -o preseeded.iso -r -J -no-emul-boot -boot-load-size 4 -boot-info-table -b isolinux.bin -c isolinux.cat ./cd
