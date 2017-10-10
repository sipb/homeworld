#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

FLANNEL_VER="0.9.0"
REVISION="1"
VERSION="${FLANNEL_VER}-${REVISION}"

DEBVER="stretch.20171009T161948Z"
BUILDVER="stretch.20171009T161948Z"
UPDATE_TIMESTAMP="2017-10-09T20:55:00-0400"

common_setup

# build flannel

init_builder
build_with_go

GODIR="${BUILDDIR}/gosrc"
rm -rf "${GODIR}"
COREOS="${GODIR}/src/github.com/coreos"
mkdir -p "${COREOS}"

tar -C "${COREOS}" -xf "${UPSTREAM}/flannel-${FLANNEL_VER}.tar.xz" "flannel-${FLANNEL_VER}/"
mv "${COREOS}/flannel-${FLANNEL_VER}" -T "${COREOS}/flannel"

build_at_path "${COREOS}/flannel"

run_builder "go build -o dist/flanneld -ldflags '-X github.com/coreos/flannel/version.Version=${FLANNEL_VER}'"

# build container

start_acbuild_from "debian-mini" "${DEBVER}"
$ACBUILD copy "${COREOS}/flannel/dist/flanneld" /usr/bin/flanneld
add_packages_to_acbuild iptables
$ACBUILD set-exec -- /usr/bin/flanneld
finish_acbuild
