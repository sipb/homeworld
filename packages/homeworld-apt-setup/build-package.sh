#!/bin/bash
set -eu

BIN=../binaries
VERSION="$(head -n 1 debian/changelog | cut -d '(' -f 2 | cut -d ')' -f 1)"
mkdir -p "${BIN}"
sbuild -d "stretch"
mv "../homeworld-apt-setup_${VERSION}_amd64.deb" -t "${BIN}"
