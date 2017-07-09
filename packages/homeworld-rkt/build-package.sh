#!/bin/bash
set -eu

BIN=../binaries
UPSTREAM=../../upstream
DVERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"
VERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d '-' -f 1)"
mkdir -p "${BIN}"
cp "${UPSTREAM}/rkt-${VERSION}.tar.xz" "rkt-${VERSION}.tar.xz"
cp "${UPSTREAM}/qemu-2.8.0.tar.xz" "qemu-2.8.0.tar.xz"
cp "${UPSTREAM}/coreos_restructured.cpio.gz" "coreos_restructured.cpio.gz"
cp "${UPSTREAM}/linux-4.9.2.tar.xz" "linux-4.9.2.tar.xz"
if [ ! -e "../go-bin-1.8.3.tgz" ]
then
	echo "No compiled go binaries found." 1>&2
	exit 1
fi
cp "../go-bin-1.8.3.tgz" "go-bin-1.8.3.tgz"
rm -f "../homeworld-rkt_${VERSION}.orig.tar.xz"
ln -s "$(basename $(pwd))/rkt-${VERSION}.tar.xz" "../homeworld-rkt_${VERSION}.orig.tar.xz"
unset GOROOT
sbuild -d stretch
echo "sbuild finished!"
mv "../homeworld-rkt_${DVERSION}_amd64.deb" -t "${BIN}"
