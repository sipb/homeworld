#!/bin/bash
set -e -u

rm -rf go acbuild
tar -xf go-bin-1.8.3.tgz go/
tar -xf acbuild-bin-0.4.0.tgz acbuild/
export GOROOT=$(pwd)/go/
export PATH="$PATH:$GOROOT/bin:$(pwd)/acbuild"

if [ "$(go version 2>/dev/null)" != "go version go1.8.3 linux/amd64" ]
then
	echo "go version mismatch! expected 1.8.3" 1>&2
	go version 1>&2
	exit 1
fi

ROOT="$(pwd)"
GODIR="${ROOT}/gosrc/"
rm -rf "${GODIR}"
COREOS="${GODIR}/src/github.com/coreos"
mkdir -p "${COREOS}"

export GOPATH="${GODIR}"

(cd "${COREOS}" && tar -xf "${ROOT}/flannel-${VERSION}.tar.xz" "flannel-${VERSION}/" && mv -T "flannel-${VERSION}" "flannel")

cd "${COREOS}/flannel"

go build -o dist/flanneld -ldflags "-X github.com/coreos/flannel/version.Version=${VERSION}"

cd "${ROOT}"

cp "${COREOS}/flannel/dist/flanneld" flanneld
# grab libraries out of the build chroot

LIBOUT="${COREOS}/flannel/dist/lib"

rm -rf "${LIBOUT}"
rm -rf "${LIBOUT}64"
mkdir -p "${LIBOUT}/x86_64-linux-gnu"
mkdir -p "${LIBOUT}64"

cp "/lib/x86_64-linux-gnu/libpthread.so.0" -t "${LIBOUT}/x86_64-linux-gnu/"
cp "/lib/x86_64-linux-gnu/libc.so.6" -t "${LIBOUT}/x86_64-linux-gnu/"
cp "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2" -t "${LIBOUT}64/"

BUILDDIR=. BINARYDIR="${COREOS}/flannel/dist/" ./build-aci "${VERSION}"

echo "flannel built!"
