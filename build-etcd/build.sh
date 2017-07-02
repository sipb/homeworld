#!/bin/bash
set -e -u
VERSION=3.1.7

cd $(dirname $0)
HYBIN=$(pwd)/../binaries/
mkdir -p ${HYBIN}

tar -xf etcd-${VERSION}.tar.xz etcd-${VERSION}/
cd etcd-${VERSION}
./build
../build-aci ${VERSION}
cp bin/etcdctl ${HYBIN}/
cp bin/etcd-${VERSION}-linux-amd64.aci ${HYBIN}/
rm -f ${HYBIN}/etcd-current-linux-amd64.aci
ln -s etcd-${VERSION}-linux-amd64.aci ${HYBIN}/etcd-current-linux-amd64.aci

FPMOPT="-s dir -t deb"
FPMOPT="$FPMOPT -n hyades-etcd -v ${VERSION} --iteration 1"
FPMOPT="$FPMOPT --maintainer 'sipb-hyades-root@mit.edu'"
FPMOPT="$FPMOPT --license APLv2 -a x86_64 --url https://github.com/coreos/etcd/"
FPMOPT="$FPMOPT ${HYBIN}/etcd-${VERSION}-linux-amd64.aci=/usr/lib/hyades/ ${HYBIN}/etcd-current-linux-amd64.aci=/usr/lib/hyades/ ${HYBIN}/etcdctl=/usr/bin/etcdctl"

fpm --vendor 'MIT SIPB Hyades Project' $FPMOPT
cp hyades-etcd_${VERSION}-1_amd64.deb ${HYBIN}/

echo "etcd built!"
