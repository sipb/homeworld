#!/bin/bash
set -e -u

GO_VER="1.8.4"

rm -rf go
tar -xf go-bin-${GO_VER}.tgz go/
export GOROOT="$(pwd)/go/"
export PATH="$PATH:$GOROOT/bin"

if [ "$(go version 2>/dev/null)" != "go version go${GO_VER} linux/amd64" ]
then
	echo "go version mismatch! expected ${GO_VER}" 1>&2
	go version 1>&2
	exit 1
fi

GODIR="$(pwd)/gosrc/"
rm -rf "${GODIR}"
ROOT=$(pwd)
mkdir "${GODIR}"

export GOPATH="${GODIR}:$(pwd)"

(cd "${GODIR}" && tar -xf "${ROOT}/golang-x-crypto.tar.xz" src)
(cd "${GODIR}" && tar -xf "${ROOT}/gopkg.in-yaml.v2.tar.xz" src)

go build src/keyserver/main/keyserver.go
go build src/keygateway/main/keygateway.go
go build src/keyclient/main/keyclient.go
go build src/keygen/main/keygen.go
go build src/keyinitadmit/main/keyinitadmit.go
go build src/keyreq/main/keyreq.go

echo "keysystem built!"
