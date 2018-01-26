#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

DNS_VER="1.14.8"
REVISION="2"
VERSION="${DNS_VER}-${REVISION}"

DNSMASQ_VER="2.78-1"
BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup

# build dnsmasq-nanny
# based on https://github.com/kubernetes/dns builds

init_builder
build_with_go

GODIR="${BUILDDIR}"
tar -C "${BUILDDIR}" -xf "${UPSTREAM}/kubernetes-dns-${DNS_VER}.tar.xz" "dns-${DNS_VER}"
mkdir -p "${GODIR}/src/k8s.io/"
mv "${BUILDDIR}/dns-${DNS_VER}" -T "${GODIR}/src/k8s.io/dns"

run_builder "go build k8s.io/dns/cmd/dnsmasq-nanny"

# build container

start_acbuild_from "dnsmasq" "${DNSMASQ_VER}"
$ACBUILD run -- mkdir -p /etc/k8s/dns/dnsmasq-nanny
$ACBUILD copy "${BUILDDIR}/dnsmasq-nanny" /usr/bin/dnsmasq-nanny
$ACBUILD set-exec -- /usr/bin/dnsmasq-nanny
finish_acbuild
