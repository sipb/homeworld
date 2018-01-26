#!/bin/bash
set -e -u

GO_VER=1.8.6

rm -rf go acbuild
tar -xf "go-bin-${GO_VER}.tgz" go/
tar -xf acbuild-bin-0.4.0.tgz acbuild/
export GOROOT="$(pwd)/go/"
export PATH="$PATH:$GOROOT/bin:$(pwd)/acbuild"

if [ "$(go version 2>/dev/null)" != "go version go${GO_VER} linux/amd64" ]
then
	echo "go version mismatch! expected ${GO_VER}" 1>&2
	go version 1>&2
	exit 1
fi

tar -xf "etcd-${VERSION}.tar.xz" "etcd-${VERSION}/"
cd "etcd-${VERSION}"
./build
cp ../local-hosts local-hosts
../build-aci "${VERSION}"
cp bin/etcdctl ..
cp "bin/etcd-${VERSION}-linux-amd64.aci" ..
cd ..
ln -s "etcd-${VERSION}-linux-amd64.aci" "etcd-current-linux-amd64.aci"

echo "etcd built!"
