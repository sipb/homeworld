#!/bin/bash
set -eu

#CHECKSUM="146593fed9dc04b2bb5c194ab0bce7737ee67c04e47b044259ed0a1cdf9743b6986ef5323f959defafce605ddfea1d0acfe91d998d8f05a6f9c7186834532fde"

BIN=../binaries
VERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"
mkdir -p "${BIN}"
sbuild -d "stretch"
mv "../homeworld-services_${VERSION}_amd64.deb" -t "${BIN}"
#echo "${CHECKSUM}  ${BIN}/homeworld-services_${VERSION}_amd64.deb" | sha512sum --check
