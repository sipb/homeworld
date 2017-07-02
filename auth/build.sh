#!/bin/bash
set -e

cd $(dirname $0)

rm -rf goroot
mkdir goroot
(cd goroot && tar -xf ../golang-x-crypto-5ef0053f77724838734b6945dd364d3847e5de1d.tar.xz src)
GOPATH=$(pwd)/goroot go build hyauth.go

./package.sh

echo "Build complete!"
