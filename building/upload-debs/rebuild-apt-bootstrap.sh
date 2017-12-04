#!/bin/bash
set -e -u

if [ "${KUID:-}" = "" ]
then
    echo "KUID expected."
    exit 1
fi

cd "$(dirname "$0")"
VERSION="$(dpkg-parsechangelog -l"../build-debs/homeworld-apt-setup/debian/changelog" -S Version)"
DEB="../build-debs/binaries/homeworld-apt-setup_${VERSION}_amd64.deb"
cp "${DEB}" /mit/hyades/homeworld-apt-setup.deb
"${GPG:-gpg}" --detach-sign --armor --local-user "${KUID}" "${DEB}" >/mit/hyades/homeworld-apt-setup.deb.asc

echo "signed!"
