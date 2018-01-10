#!/bin/bash
set -e -u

source ../setup-apt-branch/setup-apt-branch.sh
HOMEWORLD_APT_SIGNING_KEY="$(get_apt_signing_key)"

sed "s;\\\${APT_SIGNING_KEY};${HOMEWORLD_APT_SIGNING_KEY};g" < conf/distributions.in > conf/distributions

cd "$(dirname "$0")"
reprepro -Vb . update
reprepro -Vb . includedeb homeworld "../build-debs/binaries/${HOMEWORLD_APT_BRANCH}"/homeworld-*.deb
rsync -av --progress --delete-delay ./dists ./pool "/mit/hyades/apt/${HOMEWORLD_APT_BRANCH}/"

