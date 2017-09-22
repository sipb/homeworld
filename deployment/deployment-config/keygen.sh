#!/bin/bash
set -e -u

# we need a binary packaged into the keysystem
# TODO: package up the keysystem better so that we don't need to do this hacky stuff

cd "$(dirname "$0")"
BUILD_DEBS=../../building/build-debs

PACKAGE_VER="$(head -n 1 "${BUILD_DEBS}"/homeworld-keysystem/debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"

if [ -e "authorities.tgz" ]
then
    echo "authorities.tgz already found" 1>&2
    exit 1
fi

if [ -e "certgen" ]
then
    echo "certgen folder already found" 1>&2
    exit 1
fi

mkdir temp-keysystem

dpkg-deb -x "${BUILD_DEBS}/binaries/homeworld-keysystem_${PACKAGE_VER}_amd64.deb" temp-keysystem

mkdir certgen

echo "generating keys... (this may take a while; be patient!)"

temp-keysystem/usr/bin/keygen confgen/keyserver.yaml certgen/ supervisor-nodes

echo "compressing..."

tar -C certgen -czf authorities.tgz .

rm -rf temp-keysystem certgen
rm -rf certgen

echo "generated!"
