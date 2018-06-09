#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
importacbuild
upstream "etcd-${VERSION}.tar.xz"
exportorig "etcd-${VERSION}.tar.xz"
build
