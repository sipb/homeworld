#!/bin/bash
set -e -u

GO_VER=1.9.3

rm -rf go
tar -xf "go-bin-${GO_VER}.tgz" go/
export GOROOT="$(pwd)/go/"
export PATH="$PATH:$GOROOT/bin"

if [ "$(go version 2>/dev/null)" != "go version go${GO_VER} linux/amd64" ]
then
	echo "go version mismatch! expected ${GO_VER}" 1>&2
	go version 1>&2
	exit 1
fi

PROMU_VER="sipb-0.1.1"

export GOPATH="$(pwd)/gopath/"

XBIN="$(pwd)/bin"
rm -rf "${XBIN}"

rm -rf "${GOPATH}"
PROMETHEUS="${GOPATH}/src/github.com/prometheus"
mkdir -p "${PROMETHEUS}"
tar -C "${PROMETHEUS}" -xf "promu-${PROMU_VER}.tar.xz" "promu-${PROMU_VER}"
mv "${PROMETHEUS}/promu-${PROMU_VER}" "${PROMETHEUS}/promu"
tar -C "${PROMETHEUS}" -xf "prometheus-node-exporter-${VERSION}.tar.xz" "node_exporter-${VERSION}"
mv "${PROMETHEUS}/node_exporter-${VERSION}" "${PROMETHEUS}/node_exporter"

mkdir "${XBIN}"
go install github.com/prometheus/promu
cd "${PROMETHEUS}/node_exporter" && "${GOPATH}/bin/promu" build --prefix="${XBIN}"

echo "prometheus-node-exporter built!"
