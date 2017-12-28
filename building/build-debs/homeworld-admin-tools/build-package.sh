#!/bin/bash
set -e -u
source ../common/package-build-helpers.sh

function cleanup {
    rm -f resources/GIT_VERSION
}
trap cleanup EXIT

git rev-parse HEAD | tr -d '\n' >resources/GIT_VERSION
[[ -n "$(git status --porcelain)" ]] && echo -n "-dirty" >>resources/GIT_VERSION
echo >>resources/GIT_VERSION

upstream "debian-9.2.0-amd64-mini.iso"
build
