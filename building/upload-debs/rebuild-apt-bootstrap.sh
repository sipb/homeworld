#!/bin/bash
set -e -u

if [ "${KUID:-}" = "" ]
then
    echo "KUID expected."
    exit 1
fi

source ../setup-apt-branch/setup-apt-branch.sh

cd "$(dirname "$0")"
VERSION="$(dpkg-parsechangelog -l"../build-debs/homeworld-apt-setup/debian/changelog" -S Version)"
DEB="../build-debs/binaries/${HOMEWORLD_APT_BRANCH}/homeworld-apt-setup_${VERSION}_amd64.deb"
cp "${DEB}" "/mit/hyades/apt/${HOMEWORLD_APT_BRANCH}/homeworld-apt-setup.deb"
"${GPG:-gpg}" --detach-sign --armor --local-user "${KUID}" "${DEB}" >"/mit/hyades/apt/${HOMEWORLD_APT_BRANCH}/homeworld-apt-setup.deb.asc"

echo "signed!"
