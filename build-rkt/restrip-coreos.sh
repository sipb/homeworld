#!/bin/bash
set -e -u
cd $(dirname $0)
cat $(find rkt-1.27.0/ -name *.manifest | grep amd64) | sort -u >coreos-manifest.txt
zcat coreos_production_pxe_image.cpio.gz | cpio -i --to-stdout usr.squashfs >coreos_squashfs
rm -rf coreos_minimal_dir
unsquashfs -d coreos_minimal_dir -e coreos-manifest.txt coreos_squashfs
rm coreos_squashfs
cd coreos_minimal_dir
mksquashfs ./ ../coreos_resquash -root-owned -noappend
cd ..
rm -rf coreos_minimal_dir
mkdir -p coreos_ncpio/etc
cd coreos_ncpio
mv ../coreos_resquash usr.squashfs
(echo .; echo etc; echo usr.squashfs) | cpio -o | gzip -c >../coreos_restructured.cpio.gz
cd ..
rm -rf coreos_ncpio
rm coreos-manifest.txt
