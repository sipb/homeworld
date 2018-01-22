#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

KSM_VER="1.2.0"
REVISION="1"
VERSION="${KSM_VER}-${REVISION}"

BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup

# build flannel

init_builder
build_with_go

GODIR="${BUILDDIR}"
tar -C "${BUILDDIR}" -xf "${UPSTREAM}/kube-state-metrics-${KSM_VER}.tar.xz" "kube-state-metrics-${KSM_VER}/"
mkdir -p "${GODIR}/src/k8s.io/"
mv "${BUILDDIR}/kube-state-metrics-${KSM_VER}/" -T "${GODIR}/src/k8s.io/kube-state-metrics/"

run_builder "cd src/k8s.io/kube-state-metrics && make build"

# build container

start_acbuild
$ACBUILD copy "${GODIR}/src/k8s.io/kube-state-metrics/kube-state-metrics" /usr/bin/kube-state-metrics
$ACBUILD set-exec -- /usr/bin/kube-state-metrics
finish_acbuild
