#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.9"

UPDATE_TIMESTAMP="2018-01-17T17:12:00-0500"

common_setup
build_with_go

export GOPATH="${B}/godir"
mkdir "${GOPATH}"
cp -R "src" -t "${GOPATH}"

extract_upstream-as kubernetes-src-v1.9.2.tar.xz staging/src/k8s.io/ "${GOPATH}/src/k8s.io"
extract_upstream-as kubernetes-src-v1.9.2.tar.xz vendor/k8s.io/kube-openapi/pkg/common "${GOPATH}/src/k8s.io/kube-openapi/pkg/common"
extract_upstream-as kubernetes-src-v1.9.2.tar.xz vendor/github.com/ "${GOPATH}/src/github.com"
extract_upstream-as kubernetes-src-v1.9.2.tar.xz vendor/golang.org/ "${GOPATH}/src/golang.org"
extract_upstream-as kubernetes-src-v1.9.2.tar.xz vendor/gopkg.in/ "${GOPATH}/src/gopkg.in"

(cd "${B}" &&
  CGO_ENABLED=0 go build -o flannel-monitor godir/src/flannel-monitor/main/flannel-monitor.go &&
  CGO_ENABLED=0 go build -o flannel-monitor-reflector godir/src/flannel-monitor-reflector/main/flannel-monitor-reflector.go &&
  CGO_ENABLED=0 go build -o flannel-monitor-collector godir/src/flannel-monitor-collector/main/flannel-monitor-collector.go)

# build container

start_acbuild
$ACBUILD copy "${B}/flannel-monitor" /usr/bin/flannel-monitor
$ACBUILD copy "${B}/flannel-monitor-collector" /usr/bin/flannel-monitor-collector
$ACBUILD copy "${B}/flannel-monitor-reflector" /usr/bin/flannel-monitor-reflector
finish_acbuild
