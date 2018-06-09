#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

DNS_VER="1.14.8"
REVISION="3"
VERSION="${DNS_VER}-${REVISION}"

DNSMASQ_VER="2.78-1"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup
build_with_go

# build dnsmasq-nanny
# based on https://github.com/kubernetes/dns builds

export GOPATH="${B}/gosrc"
extract_upstream_as "kubernetes-dns-${DNS_VER}.tar.xz" "dns-${DNS_VER}" "${GOPATH}/src/k8s.io/dns"

(cd "${B}" && go build k8s.io/dns/cmd/dnsmasq-nanny)

# build container

start_acbuild_from "dnsmasq" "${DNSMASQ_VER}"
$ACBUILD run -- mkdir -p /etc/k8s/dns/dnsmasq-nanny
$ACBUILD copy "${B}/dnsmasq-nanny" /usr/bin/dnsmasq-nanny
$ACBUILD set-exec -- /usr/bin/dnsmasq-nanny
finish_acbuild
