#!/bin/bash
set -e -u

rm -rf go
tar -xf go-bin-1.8.3.tgz go/
export GOROOT="$(pwd)/go/"
export PATH="$PATH:$GOROOT/bin"

if [ "$(go version 2>/dev/null)" != "go version go1.8.3 linux/amd64" ]
then
	echo "go version mismatch! expected 1.8.3" 1>&2
	go version 1>&2
	exit 1
fi

GODIR="$(pwd)/gosrc/"
rm -rf "${GODIR}"
KUBE="${GODIR}/src/k8s.io/kubernetes"
mkdir -p "${KUBE}"
ROOT=$(pwd)

export GOPATH="${GODIR}"

(cd "${KUBE}" && tar -xf "${ROOT}/kubernetes-src-v${VERSION}.tar.xz")

(cd "${KUBE}" && patch -p1 <"${ROOT}/fetch-aci.patch")
(cd "${KUBE}" && patch -p1 <"${ROOT}/multihosts.patch")

(cd "${KUBE}" && make)

cp "${KUBE}/_output/local/bin/linux/amd64/hyperkube" hyperkube

echo "hyperkube built!"
