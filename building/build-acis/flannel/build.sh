#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/container-build-helpers.sh

FLANNEL_VER="0.8.0"
REVISION="1"
VERSION="${FLANNEL_VER}-${REVISION}"
DEBVER="stretch.2017.07.19.21"
BUILDVER="stretch.2017.07.19.21"

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

start_acbuild_from "debian-micro" "${DEBVER}"
$ACBUILD copy "${COREOS}/flannel/dist/flanneld" /usr/bin/flanneld
$ACBUILD set-exec -- /usr/bin/flanneld
finish_acbuild

