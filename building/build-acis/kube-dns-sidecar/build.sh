#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

DNS_VER="1.14.8"
REVISION="1"
VERSION="${DNS_VER}-${REVISION}"

DEBVER="stretch.20180111T215606Z"
BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup

# build kube-dns-sidecar
# based on https://github.com/kubernetes/dns builds

init_builder
build_with_go

GODIR="${BUILDDIR}"
tar -C "${BUILDDIR}" -xf "${UPSTREAM}/kubernetes-dns-${DNS_VER}.tar.xz" "dns-${DNS_VER}"
mkdir -p "${GODIR}/src/k8s.io/"
mv "${BUILDDIR}/dns-${DNS_VER}" -T "${GODIR}/src/k8s.io/dns"

run_builder "CGO_ENABLED=0 go build k8s.io/dns/cmd/sidecar"

# build container

start_acbuild
$ACBUILD copy "${BUILDDIR}/sidecar" /usr/bin/sidecar
$ACBUILD set-exec -- /usr/bin/sidecar
finish_acbuild
