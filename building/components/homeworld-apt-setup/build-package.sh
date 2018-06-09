#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

sed "s;\\\${APT_BRANCH};${HOMEWORLD_APT_BRANCH};g" < homeworld.sources.in > homeworld.sources

HOMEWORLD_APT_SIGNING_KEY="$(get_apt_signing_key)"
gpg --export "${HOMEWORLD_APT_SIGNING_KEY}" >homeworld-archive-keyring.gpg

build
