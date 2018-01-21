#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="0.1.3"

BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-17T17:12:00-0500"

common_setup

init_builder
build_with_go

GODIR="${BUILDDIR}/godir"
mkdir "${GODIR}"
cp -R "src" -t "${GODIR}"

tar -C "${BUILDDIR}" -xf "${UPSTREAM}/prometheus-2.0.0.tar.xz" prometheus-2.0.0/vendor
tar -C "${BUILDDIR}" -xf "${UPSTREAM}/prometheus-client_golang-0.9.0-pre1.tar.xz" client_golang-0.9.0-pre1/
mv "${BUILDDIR}/prometheus-2.0.0/vendor/github.com/" -t "${GODIR}/src/"
rm -rf "${GODIR}/src/github.com/prometheus/client_golang/"
mv "${BUILDDIR}/client_golang-0.9.0-pre1/" "${GODIR}/src/github.com/prometheus/client_golang"

run_builder "CGO_ENABLED=0 go build -o dns-monitor godir/src/dns-monitor.go"

# build container

start_acbuild
$ACBUILD copy "${BUILDDIR}/dns-monitor" /usr/bin/dns-monitor
$ACBUILD set-exec -- /usr/bin/dns-monitor
finish_acbuild
