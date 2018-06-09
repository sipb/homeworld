#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

upstream "knc-${VERSION}.tar.xz"
exportorig "knc-${VERSION}.tar.xz"
build
