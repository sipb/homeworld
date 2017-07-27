#!/bin/bash
set -e -u

ROOT=$(pwd)

tar -xf "${ROOT}/knc-${VERSION}.tar.xz" "knc-${VERSION}"
mv "knc-${VERSION}" knc-src

cd knc-src

./configure

make

echo "knc built!"
