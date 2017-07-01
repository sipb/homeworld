#!/bin/bash
set -e -u

VERSION=0.1.3

# basic structure
FPMOPT="-s dir -t deb"
# name and version
FPMOPT="$FPMOPT -n hyades-services -v ${VERSION} --iteration 1"
# packager
FPMOPT="$FPMOPT --maintainer 'sipb-hyades-root@mit.edu'"
# metadata
# TODO: better metadata
FPMOPT="$FPMOPT --license MIT -a x86_64 --url https://sipb.mit.edu/"
# dependencies
FPMOPT="$FPMOPT -d hyades-rkt -d hyades-etcd -d hyades-flannel -d hyades-hyperkube"
# get binary
FPMOPT="$FPMOPT wrappers/=/usr/lib/hyades/ services/=/usr/lib/systemd/system/"

fpm --vendor 'MIT SIPB Hyades Project' $FPMOPT
cp hyades-services_${VERSION}-1_amd64.deb ../binaries
