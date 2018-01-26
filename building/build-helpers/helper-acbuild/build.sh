#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

GO_VER=1.8.6
ACBUILD_VER=0.4.0
UPSTREAM="../../upstream"

if [ -e "../acbuild-bin-${ACBUILD_VER}.tgz" ]
then
	echo "Already built acbuild!"
	exit 0
fi

rm -rf go acbuild
tar -xf "../go-bin-${GO_VER}.tgz" go
tar -xf "${UPSTREAM}/acbuild-src-${ACBUILD_VER}.tgz" acbuild

sed -i "s/^VERSION=.*$/VERSION=v${ACBUILD_VER}/" acbuild/build

GLDFLAGS="-X github.com/appc/acbuild/lib.Version=v${ACBUILD_VER}"

export GOROOT="$(pwd)/go"
export PATH="$GOROOT/bin${PATH:+:$PATH}"

(cd acbuild && ./build)

rm -rf rel
mkdir -p rel/acbuild
cp acbuild/bin/* rel/acbuild

tar -C rel -czf "../acbuild-bin-${ACBUILD_VER}.tgz" acbuild
rm -rf rel

echo "Built acbuild!"
