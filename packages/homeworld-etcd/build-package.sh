#!/bin/bash
set -eu

#CHECKSUM="146593fed9dc04b2bb5c194ab0bce7737ee67c04e47b044259ed0a1cdf9743b6986ef5323f959defafce605ddfea1d0acfe91d998d8f05a6f9c7186834532fde"

BIN=../binaries
UPSTREAM=../../upstream
VERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d '-' -f 1)"
mkdir -p "${BIN}"
cp "${UPSTREAM}/etcd-${VERSION}.tar.xz" "etcd-${VERSION}.tar.xz"
if [ ! -e "../go-bin-1.8.3.tgz" ]
then
	echo "No compiled go binaries found." 1>&2
	exit 1
fi
if [ ! -e "../acbuild-bin-0.4.0.tgz" ]
then
	echo "No compiled acbuild binaries found." 1>&2
	exit 1
fi
cp "../go-bin-1.8.3.tgz" "go-bin-1.8.3.tgz"
cp "../acbuild-bin-0.4.0.tgz" "acbuild-bin-0.4.0.tgz"
rm -f "../homeworld-etcd_${VERSION}.orig.tar.xz"
ln -s "$(basename $(pwd))/etcd-${VERSION}.tar.xz" "../homeworld-etcd_${VERSION}.orig.tar.xz"
# sudo pbuilder create --distribution stretch
unset GOROOT
pdebuild --buildresult "${BIN}"
#echo "${CHECKSUM}  ${BIN}/homeworld-etcd_${VERSION}_amd64.deb" | sha512sum --check
