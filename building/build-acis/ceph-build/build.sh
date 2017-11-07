#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/container-build-helpers.sh

VERSION="stretch.20171105T160402Z"

DEBVER="${VERSION}"
UPDATE_TIMESTAMP="2017-11-05T19:27:00-0500"

common_setup

BUILD_DEPS="bc btrfs-tools cmake cpio cryptsetup-bin cython cython3 gdisk git gperf jq libaio-dev libbabeltrace-ctf-dev libbabeltrace-dev libblkid-dev libcurl4-gnutls-dev libexpat1-dev libgoogle-perftools-dev libibverbs-dev libkeyutils-dev libldap2-dev libleveldb-dev liblttng-ust-dev libleveldb-dev liblttng-ust-dev libnss3-dev libsnappy-dev libssl-dev libtool libudev-dev libxml2-dev lsb-release parted pkg-config python python-all-dev python-cherrypy3 python-nose python-pecan python-prettytable python-setuptools python-sphinx python-werkzeug python3-all-dev python3-setuptools socat uuid-runtime virtualenv xfslibs-dev xfsprogs xmlstarlet yasm zlib1g-dev"

start_acbuild_from "debian-build" "${DEBVER}"
add_packages_to_acbuild ${BUILD_DEPS}
$ACBUILD set-exec -- /bin/bash
finish_acbuild
