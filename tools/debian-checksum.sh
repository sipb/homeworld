#!/bin/bash
set -e -u

VERSION="20190702+deb10u2"
RELEASE="buster"

echo "finding hash for release ${RELEASE} version ${VERSION}:" >&2

WORK=$(mktemp -d)
# shellcheck disable=SC2064
trap "rm -r $WORK" EXIT

cd "$WORK"

curl -sO "http://debian.csail.mit.edu/debian/dists/${RELEASE}/Release"
curl -sO "http://debian.csail.mit.edu/debian/dists/${RELEASE}/Release.gpg"

echo 'verifying Release against debian-archive-keyring' >&2
gpg --no-default-keyring --keyring "/usr/share/keyrings/debian-archive-keyring.gpg" --quiet --verify Release.gpg Release 2>/dev/null >/dev/null || (echo '[failed]' >&2 && false)
echo '[success]' >&2

curl -sO "http://debian.csail.mit.edu/debian/dists/${RELEASE}/main/installer-amd64/${VERSION}/images/SHA256SUMS"

echo 'verifying SHA256SUMS against Release' >&2
# [tail] because the sha256's are after the md5's :P
echo "$(grep "main/installer-amd64/$VERSION/images/SHA256SUMS" Release | tail -n 1 | cut -d " " -f 2)" SHA256SUMS | sha256sum --check --strict >&2

echo >&2
echo "new version block:" >&2

echo >&2
echo "VERSION = '${VERSION}'"
echo "RELEASE = '${RELEASE}'"
echo "MINI_ISO_HASH = '$(grep './netboot/mini.iso' SHA256SUMS | cut -d " " -f 1)'"
