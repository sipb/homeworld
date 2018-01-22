#!/bin/bash
set -e -u

GO_VER="1.8.4"

if [ -e ../go-bin-${GO_VER}.tgz ]
then
	echo "Already built go!"
	exit 0
fi

ROOT="$(pwd)"
rm -rf go go1.4 go${GO_VER}
tar -xf go1.4-bootstrap-20170531.tar.xz go
mv go go1.4
tar -xf go${GO_VER}.src.tar.xz go
mv go go${GO_VER}
BOOTSTRAP="${ROOT}/go1.4"
cd "${ROOT}/go1.4/src/"
./make.bash
cd "${ROOT}/go${GO_VER}/src"
GOROOT_FINAL="/usr/lib/homeworld-goroot" GOARCH="amd64" GOOS="linux" CGO_ENABLED="1" GOROOT_BOOTSTRAP="${BOOTSTRAP}" ./make.bash
cd "${ROOT}"
rm -rf go1.4
mv go${GO_VER} go
echo "renamed go${GO_VER}/ -> go/"
tar -czf ../go-bin-${GO_VER}.tgz go/

echo "golang built!"
