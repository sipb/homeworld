#!/bin/bash

# restrip-coreos.sh exists because coreos distributions are rather big, and
# we'd rather just have the crucial parts, rather than everything, especially
# for storage in git.
# it will take in a production image and spit out a restructured and much, much
# smaller image.

set -e -u
cd "$(dirname "$0")"
find rkt-1.29.0/ -name '*.manifest' -print0 | grep -z amd64 | xargs -0 cat -- | sort -u >coreos-manifest.txt
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
