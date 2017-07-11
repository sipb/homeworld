#!/usr/bin/env bash
set -e

cd "$(dirname $0)"

if [ -e "../acbuild-bin-0.4.0.tgz" ]
then
	echo "Already built acbuild!"
	exit 0
fi

rm -rf go acbuild
tar -xf go-bin-1.8.3.tgz go
tar -xf acbuild-src-0.4.0.tgz acbuild

VERSION=v0.4.0

sed -i "s/^VERSION=.*$/VERSION=${VERSION}/" acbuild/build

GLDFLAGS="-X github.com/appc/acbuild/lib.Version=${VERSION}"

export GOROOT="$(pwd)/go"

cd acbuild

./build

cd ..

rm -rf rel
mkdir -p rel/acbuild
cp acbuild/bin/* rel/acbuild

(cd rel && tar -czf "../../acbuild-bin-0.4.0.tgz" acbuild)

echo "Built acbuild!"
