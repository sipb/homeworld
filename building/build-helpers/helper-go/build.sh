#!/bin/bash
set -e -u

if [ -e ../go-bin-1.8.3.tgz ]
then
	echo "Already built go!"
	exit 0
fi

ROOT="$(pwd)"
rm -rf go go1.4 go1.8.3
tar -xf go1.4-bootstrap-20170531.tar.xz go
mv go go1.4
tar -xf go1.8.3.src.tar.xz go
mv go go1.8.3
BOOTSTRAP="${ROOT}/go1.4"
cd "${ROOT}/go1.4/src/"
./make.bash
cd "${ROOT}/go1.8.3/src"
GOROOT_FINAL="/usr/lib/homeworld-goroot" GOARCH="amd64" GOOS="linux" CGO_ENABLED="1" GOROOT_BOOTSTRAP="${BOOTSTRAP}" ./make.bash
cd "${ROOT}"
rm -rf go1.4
mv go1.8.3 go
echo "renamed go1.8.3/ -> go/"
tar -czf ../go-bin-1.8.3.tgz go/

echo "golang built!"
