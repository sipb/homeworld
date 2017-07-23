#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "golang-x-crypto-5ef0053f77724838734b6945dd364d3847e5de1d.tar.xz" "golang-x-crypto.tar.xz"
build
