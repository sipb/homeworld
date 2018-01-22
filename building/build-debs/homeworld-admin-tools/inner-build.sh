#!/bin/bash
set -e -u

rm -rf spire spire.zip src/resources src/__pycache__ resources/__pycache__
cp -RT resources src/resources

if [[ -v VERSION ]]
then
    echo "$VERSION" >src/resources/DEB_VERSION
else
    echo "not a debian build" >src/resources/DEB_VERSION
fi

if [[ ! -e src/resources/GIT_VERSION ]]
then
    git rev-parse HEAD | tr -d '\n' >src/resources/GIT_VERSION
    [[ -n "$(git status --porcelain)" ]] && echo -n "-dirty" >>src/resources/GIT_VERSION
    echo >>src/resources/GIT_VERSION
fi

if [[ ! -e src/resources/APT_BRANCH ]]
then
    source ../../setup-apt-branch/setup-apt-branch.sh
    echo "${HOMEWORLD_APT_BRANCH}" >src/resources/APT_BRANCH
    HOMEWORLD_APT_SIGNING_KEY="$(get_apt_signing_key)"
    gpg --export "${HOMEWORLD_APT_SIGNING_KEY}" >src/resources/homeworld-archive-keyring.gpg
fi

(cd src && zip -r ../spire.zip *)

python3 spire.zip iso regen-cdpack debian-9.2.0-amd64-mini.iso src/resources/debian-9.2.0-cdpack.tgz

rm spire.zip
(cd src && zip -r ../spire.zip *)
rm -rf src/resources
echo "#!/usr/bin/env python3" | cat - spire.zip >spire
chmod +x spire

echo "admin-tools built!"
