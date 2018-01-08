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
