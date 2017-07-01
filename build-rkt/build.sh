#!/bin/bash
set -e -u

cd $(dirname $0)

rm -rf rkt-1.26.0/
tar -xf rkt-1.26.0.tar.xz rkt-1.26.0/

patch -p0 <rkt.patch

if gcc --version | grep -q 'Debian 4.9'
then
	rm rkt-1.26.0/stage1/usr_from_kvm/kernel/patches/0002-for-debian-gcc.patch
fi

sha512sum --check <<EOF
85adf3715cba4a457efea8359ebed34413ac63ee58fe920c5713501dec1e727e167416e9d67a9e2d9430aa9f3a53ad0ac26a4f749984bc5a3f3c37ac504f75de  linux-4.9.2.tar.xz
10e3fcd515b746be2af55636a6dd9334706cc21ab2bb8b2ba38fe342ca51b072890fd1ae2cd8777006036b0a08122a621769cb5903ab16cfbdcc47c9444aa208  qemu-2.8.1.1.tar.xz
EOF

cp coreos_restructured.cpio.gz rkt-1.26.0/coreos_production_pxe_image.cpio.gz
mkdir -p rkt-1.26.0/build-rkt-1.26.0/tmp/usr_from_kvm/kernel/
cp linux-4.9.2.tar.xz rkt-1.26.0/build-rkt-1.26.0/tmp/usr_from_kvm/kernel/linux-4.9.2.tar.xz
mkdir -p rkt-1.26.0/build-rkt-1.26.0/tmp/usr_from_kvm/qemu/
cp qemu-2.8.1.1.tar.xz rkt-1.26.0/build-rkt-1.26.0/tmp/usr_from_kvm/qemu/qemu-2.8.1.1.tar.xz
cd rkt-1.26.0/

./autogen.sh

./configure \
	--disable-tpm --prefix=/usr \
	--with-stage1-flavors=coreos,kvm \
	--with-stage1-default-name=hyades.mit.edu/rkt/stage1-coreos \
	--with-stage1-default-version=1.26.0 \
	--with-coreos-local-pxe-image-path=coreos_production_pxe_image.cpio.gz \
	--with-coreos-local-pxe-image-systemd-version=v231 \
	--with-stage1-default-images-directory=/usr/lib/rkt/stage1-images \
	--with-stage1-default-location=/usr/lib/rkt/stage1-images/stage1-coreos.aci

# make manpages
# make bash-completion
make -j4

cd ..

BUILDDIR=rkt-1.26.0/build-rkt-1.26.0/
BUILDDIR=${BUILDDIR} ./build-pkgs.sh 1.26.0

cp ${BUILDDIR}/target/bin/hyades-rkt_1.26.0-1_amd64.deb ../binaries/
