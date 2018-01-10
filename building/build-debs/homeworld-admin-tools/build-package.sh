#!/bin/bash
set -e -u
source ../common/package-build-helpers.sh

function cleanup {
    rm -f resources/GIT_VERSION
    rm -f resources/APT_BRANCH
}
trap cleanup EXIT

git rev-parse HEAD | tr -d '\n' >resources/GIT_VERSION
[[ -n "$(git status --porcelain)" ]] && echo -n "-dirty" >>resources/GIT_VERSION
echo >>resources/GIT_VERSION

echo "${HOMEWORLD_APT_BRANCH}" >resources/APT_BRANCH

HOMEWORLD_APT_SIGNING_KEY="$(get_apt_signing_key)"
gpg --export "${HOMEWORLD_APT_SIGNING_KEY}" >resources/homeworld-archive-keyring.gpg

upstream "debian-9.2.0-amd64-mini.iso"
build
