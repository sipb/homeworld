#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

VERSION="${1:-}"
TARGET="../platform/spire/debian-iso/deps.bzl"

if [ "$VERSION" = "" ]
then
    echo "usage: $0 20190702+deb10u3 [or other new version]" 1>&2
    echo "usage: $0 auto" 1>&2
    echo 1>&2
    echo "this command will try to update debian-iso/deps.bzl and then make a commit" 1>&2
    exit 1
fi

## discover the latest version, if relevant

if [ "$VERSION" = "auto" ]
then
    # yes, this is a HTTP fetch, and in theory could be MITM'd, but if the version number is
    # invalid, debian-iso-checksum.sh will just fail later.
    VERSION="$(curl -s http://debian.csail.mit.edu/debian/dists/buster/Release |
               grep -E "^ [0-9a-f]{64} [ 0-9]+ main/installer-amd64/[0-9]+[+][0-9a-z]+/images/SHA256SUMS$" |
               sed 's|^[^/]*/installer-amd64/\([^/]*\)/.*|\1|g')"
fi

## make sure the git state is clean enough for us

if ! git diff --quiet "${TARGET}" || ! git diff --cached --quiet "${TARGET}"
then
    echo "cannot update ${TARGET} automatically; it has already been modified" 1>&2
    exit 1
fi

## make the change

./debian-iso-checksum.sh "${VERSION}" | ./update-defs.py "${TARGET}"

## make the commit

if git diff --quiet "${TARGET}"
then
    echo "no commit to make; files did not change" 1>&2
    exit 1
fi

git commit -e -v "${TARGET}" -m "debian-iso: bump version to $VERSION" -m "This commit was automatically created with bump-debian-iso.sh."
