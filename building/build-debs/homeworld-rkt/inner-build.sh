#!/bin/bash
set -e -u

GO_VER="1.8.4"

rm -rf go
tar -xf go-bin-${GO_VER}.tgz go/
export GOROOT="$(pwd)/go/"
export PATH="$PATH:$GOROOT/bin"

if [ "$(go version 2>/dev/null)" != "go version go${GO_VER} linux/amd64" ]
then
	echo "go version mismatch! expected ${GO_VER}" 1>&2
	go version 1>&2
	exit 1
fi

rm -rf rkt-1.27.0/
tar -xf rkt-1.27.0.tar.xz rkt-1.27.0/
patch -p0 <rkt.patch

if gcc --version | grep -q 'Debian 4.9'
then
	rm rkt-1.27.0/stage1/usr_from_kvm/kernel/patches/0002-for-debian-gcc.patch
fi

cp coreos_restructured.cpio.gz rkt-1.27.0/coreos_production_pxe_image.cpio.gz
mkdir -p rkt-1.27.0/build-rkt-1.27.0/tmp/usr_from_kvm/kernel/
cp linux-4.9.2.tar.xz -t rkt-1.27.0/build-rkt-1.27.0/tmp/usr_from_kvm/kernel/
mkdir -p rkt-1.27.0/build-rkt-1.27.0/tmp/usr_from_kvm/qemu/
cp qemu-2.8.0.tar.xz -t rkt-1.27.0/build-rkt-1.27.0/tmp/usr_from_kvm/qemu/

cd rkt-1.27.0/

./autogen.sh

./configure \
	--disable-tpm --prefix=/usr \
	--with-stage1-flavors=coreos,kvm \
	--with-stage1-default-flavor=kvm \
	--with-coreos-local-pxe-image-path=coreos_production_pxe_image.cpio.gz \
	--with-coreos-local-pxe-image-systemd-version=v231 \
	--with-stage1-default-images-directory=/usr/lib/rkt/stage1-images \
	--with-stage1-default-location=/usr/lib/rkt/stage1-images/stage1-kvm.aci

make -j4

cd ..

echo "rkt built!"
