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

(cd src && zip -r ../spire.zip *)

python3 spire.zip iso regen-cdpack debian-9.2.0-amd64-mini.iso src/resources/debian-9.2.0-cdpack.tgz

rm spire.zip
(cd src && zip -r ../spire.zip *)
rm -rf src/resources
echo "#!/usr/bin/env python3" | cat - spire.zip >spire
chmod +x spire

echo "admin-tools built!"
