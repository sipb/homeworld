#!/bin/bash
set -eu

#CHECKSUM="146593fed9dc04b2bb5c194ab0bce7737ee67c04e47b044259ed0a1cdf9743b6986ef5323f959defafce605ddfea1d0acfe91d998d8f05a6f9c7186834532fde"

BIN=../binaries
UPSTREAM=../../upstream
VERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"
mkdir -p "${BIN}"
if [ ! -e "../go-bin-1.8.3.tgz" ]
then
	echo "No compiled go binaries found." 1>&2
	exit 1
fi
cp "../go-bin-1.8.3.tgz" "go-bin-1.8.3.tgz"
cp "${UPSTREAM}/golang-x-crypto-5ef0053f77724838734b6945dd364d3847e5de1d.tar.xz" "golang-x-crypto.tar.xz"
rm -f "../homeworld-authserver_${VERSION}.orig.tar.xz"
unset GOROOT
sbuild -d stretch
mv "../homeworld-authserver_${VERSION}_amd64.deb" -t "${BIN}"
#echo "${CHECKSUM}  ${BIN}/homeworld-authserver_${VERSION}_amd64.deb" | sha512sum --check
