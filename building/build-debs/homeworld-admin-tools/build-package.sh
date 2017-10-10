#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

upstream "debian-9.2.0-amd64-mini.iso"
build
