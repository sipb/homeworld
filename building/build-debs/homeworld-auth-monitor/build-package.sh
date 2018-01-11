#!/bin/bash
set -eu
source ../common/package-build-helpers.sh

importgo
upstream "prometheus-2.0.0.tar.xz"
upstream "prometheus-client_golang-0.9.0-pre1.tar.xz"
upstream "golang-x-crypto-5ef0053f77724838734b6945dd364d3847e5de1d.tar.xz" "golang-x-crypto.tar.xz"
upstream "gopkg.in-yaml.v2-eb3733d160e74a9c7e442f435eb3bea458e1d19f.tar.xz" "gopkg.in-yaml.v2.tar.xz"
for upkg in keycommon util
do
	rm -rf "src/$upkg"
	cp -R "../homeworld-keysystem/src/$upkg" -T "src/$upkg"
done
build
