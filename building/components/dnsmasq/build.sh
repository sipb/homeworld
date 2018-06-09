#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

DNSMASQ_VER="2.78"
REVISION="1"
VERSION="${DNSMASQ_VER}-${REVISION}"

DEBVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-01-11T22:47:00-0500"

common_setup

# build dnsmasq
# based on https://github.com/kubernetes/dns builds

extract_upstream_as "dnsmasq-${DNSMASQ_VER}.tar.xz" "dnsmasq-${DNSMASQ_VER}/" "${B}/dnsmasq"

mkdir -p "${B}/run"

(cd "${B}/dnsmasq" && make)

# build container

start_acbuild_from "debian-micro" "${DEBVER}"
$ACBUILD copy dnsmasq.conf /etc/dnsmasq.conf
$ACBUILD copy "${B}/dnsmasq/src/dnsmasq" /usr/sbin/dnsmasq
$ACBUILD copy "${B}/run" /var/run
$ACBUILD set-exec -- /usr/sbin/dnsmasq --keep-in-foreground
finish_acbuild
