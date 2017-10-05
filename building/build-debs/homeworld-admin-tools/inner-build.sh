#!/bin/bash
set -e -u

rm -rf spire spire.zip src/resources src/__pycache__ resources/__pycache__
cp -RT resources src/resources
(cd src && zip -r ../spire.zip *)

python3 spire.zip iso regen-cdpack debian-9.0.0-amd64-mini.iso src/resources/debian-9.0.0-cdpack.tgz

rm spire.zip
(cd src && zip -r ../spire.zip *)
rm -rf src/resources
echo "#!/usr/bin/env python3" | cat - spire.zip >spire
chmod +x spire

echo "admin-tools built!"
