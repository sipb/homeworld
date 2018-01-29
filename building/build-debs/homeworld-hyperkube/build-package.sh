#!/bin/bash
set -eu
GOVER=1.8.6  # override the definition in package-build-helpers

source ../common/package-build-helpers.sh

importgo
upstream "kubernetes-src-v${VERSION}.tar.xz"
exportorig "hyperkube-${VERSION}.tar.xz"
build
