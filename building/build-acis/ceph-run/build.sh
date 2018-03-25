#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

CEPH_VER="12.2.3-1"
REVISION="9"
VERSION="${CEPH_VER}-${REVISION}"

BUILDVER="stretch.20180111T215606Z"
UPDATE_TIMESTAMP="2018-03-17T17:11:00-0400"

common_setup

# build container

start_acbuild_from "ceph" "${CEPH_VER}"
$ACBUILD copy-to-dir scripts/* /usr/bin/
add_packages_to_acbuild curl uuid-runtime
finish_acbuild
