#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "prometheus-node-exporter-${VERSION}.tar.xz"
upstream "promu-sipb-0.1.1.tar.xz"
exportorig "prometheus-node-exporter-${VERSION}.tar.xz"
build
