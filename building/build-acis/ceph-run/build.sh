#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

CEPH_VER="12.2.3-1"
REVISION="1"
VERSION="${CEPH_VER}-${REVISION}"

BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-03-11T14:25:00-0400"

common_setup

tar -C "${BUILDDIR}" -xf "${UPSTREAM}/kubernetes-dns-${DNS_VER}.tar.xz" "dns-${DNS_VER}"
mkdir -p "${GODIR}/src/k8s.io/"
mv "${BUILDDIR}/dns-${DNS_VER}" -T "${GODIR}/src/k8s.io/dns"

# build container

start_acbuild_from "ceph" "${CEPH_VER}"
$ACBUILD copy "scripts/" /usr/bin/
finish_acbuild
