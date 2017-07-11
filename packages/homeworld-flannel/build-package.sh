#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "flannel-${VERSION}.tar.xz"
exportorig "flannel-${VERSION}.tar.xz"
build
