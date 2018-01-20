#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.6"

BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-17T17:12:00-0500"

common_setup

init_builder
build_with_go

GODIR="${BUILDDIR}/godir"
mkdir "${GODIR}"
cp -R "src" -t "${GODIR}"

tar -C "${BUILDDIR}" -xf "${UPSTREAM}/kubernetes-src-v1.8.0.tar.xz" ./staging/src/k8s.io/ ./vendor/github.com/ ./vendor/golang.org/ ./vendor/gopkg.in/ ./vendor/k8s.io/kube-openapi/pkg/common
mv "${BUILDDIR}/staging/src/k8s.io" -t "${GODIR}/src/"
mv "${BUILDDIR}/vendor/k8s.io/kube-openapi" -t "${GODIR}/src/k8s.io/"
mv "${BUILDDIR}/vendor/github.com" -t "${GODIR}/src/"
mv "${BUILDDIR}/vendor/golang.org" -t "${GODIR}/src/"
mv "${BUILDDIR}/vendor/gopkg.in" -t "${GODIR}/src/"

run_builder "CGO_ENABLED=0 go build -o flannel-monitor godir/src/flannel-monitor/main/flannel-monitor.go" \
    "CGO_ENABLED=0 go build -o flannel-monitor-reflector godir/src/flannel-monitor-reflector/main/flannel-monitor-reflector.go" \
    "CGO_ENABLED=0 go build -o flannel-monitor-collector godir/src/flannel-monitor-collector/main/flannel-monitor-collector.go"

# build container

start_acbuild
$ACBUILD copy "${BUILDDIR}/flannel-monitor" /usr/bin/flannel-monitor
$ACBUILD copy "${BUILDDIR}/flannel-monitor-collector" /usr/bin/flannel-monitor-collector
$ACBUILD copy "${BUILDDIR}/flannel-monitor-reflector" /usr/bin/flannel-monitor-reflector
finish_acbuild
