#!/bin/bash
set -e -u

cd "$(dirname $0)"

GO_VER="1.9.3"
UPSTREAM="../../upstream"

if [ -e "../go-bin-${GO_VER}.tgz" ]
then
	echo "Already built go!"
	exit 0
fi

ROOT="$(pwd)"
rm -rf go go1.4 "go${GO_VER}"
tar -xf "${UPSTREAM}/go1.4-bootstrap-20170531.tar.xz" go
mv go go1.4
tar -xf "${UPSTREAM}/go${GO_VER}.src.tar.xz" go
mv go "go${GO_VER}"
BOOTSTRAP="${ROOT}/go1.4"
cd "${ROOT}/go1.4/src/"
./make.bash
cd "${ROOT}/go${GO_VER}/src"
GOROOT_FINAL="/usr/lib/homeworld-goroot" GOARCH="amd64" GOOS="linux" CGO_ENABLED="1" GOROOT_BOOTSTRAP="${BOOTSTRAP}" ./make.bash
cd "${ROOT}"
rm -rf go1.4
mv "go${GO_VER}" go
tar -czf "../go-bin-${GO_VER}.tgz" go/
rm -rf go

echo "golang ${GO_VER} built!"
