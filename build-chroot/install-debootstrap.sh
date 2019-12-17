#!/bin/bash
set -e -u

VERSION="1.0.106"
DEBI="20180811T145253Z"

cd "$(dirname "$0")"
curl -L -o "debootstrap-${VERSION}.deb" "http://snapshot.debian.org/archive/debian/${DEBI}/pool/main/d/debootstrap/debootstrap_${VERSION}_all.deb"
sha256sum --check "debootstrap-${VERSION}.deb.sha256"
sudo dpkg "$@" --install "debootstrap-${VERSION}.deb"
echo "debootstrap hold" | sudo dpkg "$@" --set-selections
rm "debootstrap-${VERSION}.deb"
