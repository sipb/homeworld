#!/bin/bash
set -e -u

rm -rf go
tar -xf go-bin-1.8.3.tgz go/
export GOROOT=$(pwd)/go/
export PATH="$PATH:$GOROOT/bin"

if [ "$(go version 2>/dev/null)" != "go version go1.8.3 linux/amd64" ]
then
	echo "go version mismatch! expected 1.8.3" 1>&2
	go version 1>&2
	exit 1
fi

GODIR="$(pwd)/gosrc/"
rm -rf "${GODIR}"
COREOS="${GODIR}/src/github.com/coreos"
mkdir -p "${COREOS}"
ROOT=$(pwd)

export GOPATH="${GODIR}"

(cd "${COREOS}" && tar -xf "${ROOT}/flannel-${VERSION}.tar.xz" "flannel-${VERSION}/" && mv -T "flannel-${VERSION}" "flannel")

(cd "${COREOS}/flannel" && CGO_ENABLED=1 make dist/flanneld)

cp "${COREOS}/flannel/dist/flanneld" flanneld

echo "flannel built!"
