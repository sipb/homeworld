#!/bin/bash
set -e -u

source ../setup-apt-branch/setup-apt-branch.sh
HOMEWORLD_APT_SIGNING_KEY="$(get_apt_signing_key)"

if ! compgen -G "../build-debs/binaries/${HOMEWORLD_APT_BRANCH}"/homeworld-apt-setup*.deb
then
    echo "error: homeworld-apt-setup not built!" 1>&2
    exit 1
fi

if ! compgen -G "../build-debs/binaries/${HOMEWORLD_APT_BRANCH}"/homeworld-admin-tools*.deb
then
    echo "error: homeworld-admin-tools not built" 1>&2
    exit 1
fi

cd "$(dirname "$0")"
mkdir -p "./apt/${HOMEWORLD_APT_BRANCH}"
sed "s;\\\${APT_SIGNING_KEY};${HOMEWORLD_APT_SIGNING_KEY};g" < ./conf/distributions.in > ./conf/distributions
cp -r -T conf "./apt/${HOMEWORLD_APT_BRANCH}/conf"
reprepro -Vb "./apt/${HOMEWORLD_APT_BRANCH}" update
reprepro -Vb "./apt/${HOMEWORLD_APT_BRANCH}" includedeb homeworld "../build-debs/binaries/${HOMEWORLD_APT_BRANCH}"/homeworld-*.deb
rsync -av --progress --delete-delay "./apt/${HOMEWORLD_APT_BRANCH}/dists" "./apt/${HOMEWORLD_APT_BRANCH}/pool" "/mit/hyades/apt/${HOMEWORLD_APT_BRANCH}/"

