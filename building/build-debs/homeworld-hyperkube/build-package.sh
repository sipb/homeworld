#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "kubernetes-src-v${VERSION}.tar.xz"
exportorig "hyperkube-${VERSION}.tar.xz"
build
