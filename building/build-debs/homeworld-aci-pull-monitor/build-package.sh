#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "prometheus-2.0.0.tar.xz"
upstream "prometheus-client_golang-0.9.0-pre1.tar.xz"
build
