#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.5"

UPDATE_TIMESTAMP="2018-01-17T17:12:00-0500"

common_setup
build_with_go

export GOPATH="${B}/godir"
mkdir "${GOPATH}"
cp -R "src" -t "${GOPATH}"

extract_upstream_as "prometheus-2.0.0.tar.xz" prometheus-2.0.0/vendor/github.com/ "${GOPATH}/src/github.com"
rm -rf "${GOPATH}/src/github.com/prometheus/client_golang/"
extract_upstream_as "prometheus-client_golang-0.9.0-pre1.tar.xz" client_golang-0.9.0-pre1/ "${GOPATH}/src/github.com/prometheus/client_golang"

(cd "${B}" && CGO_ENABLED=0 go build -o dns-monitor godir/src/dns-monitor.go)

# build container

start_acbuild
$ACBUILD copy "${B}/dns-monitor" /usr/bin/dns-monitor
$ACBUILD set-exec -- /usr/bin/dns-monitor
finish_acbuild
