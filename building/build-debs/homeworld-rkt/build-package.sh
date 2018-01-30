#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "rkt-${VERSION}.tar.xz"
upstream "qemu-2.8.0.tar.xz"
upstream "coreos_restructured-1478.0.0.cpio.gz"
upstream "linux-4.9.2.tar.xz"
exportorig "rkt-${VERSION}.tar.xz"
build
