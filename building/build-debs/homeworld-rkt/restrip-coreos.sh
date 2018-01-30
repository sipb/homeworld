#!/bin/bash

# restrip-coreos.sh exists because coreos distributions are rather big, and
# we'd rather just have the crucial parts, rather than everything, especially
# for storage in git.
# it will take in a production image and spit out a restructured and much, much
# smaller image.

set -e -u

if [ ! -e "rkt-1.29.0" ]
then
    echo "Cannot find unpacked rkt source archive in current directory." 1>&2
    exit 1
fi

if [ ! -e "coreos_production_pxe_image.cpio.gz" ]
then
    echo "Cannot find downloaded coreos_production_pxe_image.cpio.gz in current directory." 1>&2
    exit
fi

find rkt-1.29.0/ -name '*.manifest' -print0 | grep -z amd64 | xargs -0 cat -- | sort -u >coreos-manifest.txt
zcat coreos_production_pxe_image.cpio.gz | cpio -i --to-stdout usr.squashfs >coreos_squashfs
rm -rf coreos_minimal_dir
unsquashfs -d coreos_minimal_dir -e coreos-manifest.txt coreos_squashfs
rm coreos-manifest.txt coreos_squashfs
(cd coreos_minimal_dir && mksquashfs ./ ../coreos_resquash -root-owned -noappend)
rm -rf coreos_minimal_dir
mkdir -p coreos_ncpio/etc
mv coreos_resquash coreos_ncpio/usr.squashfs
(cd coreos_ncpio && ((echo .; echo etc; echo usr.squashfs) | cpio -o)) | gzip -c >coreos_restructured.cpio.gz
rm -rf coreos_ncpio

echo "output in coreos_restructured.cpio.gz"
