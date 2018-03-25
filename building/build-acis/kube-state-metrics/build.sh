#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

KSM_VER="1.2.0"
REVISION="4"
VERSION="${KSM_VER}-${REVISION}"

UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup
build_with_go

# build kube-state-metrics

export GOPATH="${B}/godir"
extract_upstream_as "kube-state-metrics-${KSM_VER}.tar.xz" "kube-state-metrics-${KSM_VER}/" "${GOPATH}/src/k8s.io/kube-state-metrics/"

(cd "${GOPATH}/src/k8s.io/kube-state-metrics" && make build)

# build container

start_acbuild
$ACBUILD copy "${GOPATH}/src/k8s.io/kube-state-metrics/kube-state-metrics" /usr/bin/kube-state-metrics
$ACBUILD set-exec -- /usr/bin/kube-state-metrics
$ACBUILD port add metrics tcp 80
$ACBUILD port add metametrics tcp 81
finish_acbuild
