#!/bin/bash
set -e -u
cd $(dirname $0)
ROOT=$(pwd)
rm -rf go/
mkdir -p go/src/github.com/coreos/
cd go/src/github.com/coreos/
echo "extracting..."
tar -xf ${ROOT}/flannel-0.7.1.tar.xz flannel-0.7.1/
echo "extracted!"
export GOPATH=${ROOT}/go
mv flannel-0.7.1/ flannel/
cd flannel/
CGO_ENABLED=1 make dist/flanneld
cd ${ROOT}
./package.sh
echo "built flannel binaries!"
