#!/bin/bash
set -e -u

# basic structure
FPMOPT="-s dir -t deb"
# name and version
FPMOPT="$FPMOPT -n hyades-hyperkube -v 1.6.4 --iteration 1"
# packager
FPMOPT="$FPMOPT --maintainer 'sipb-hyades-root@mit.edu'"
# metadata
FPMOPT="$FPMOPT --license APLv2 -a x86_64 --url https://kubernetes.io/"
# get binary
FPMOPT="$FPMOPT --prefix /usr/bin --chdir go/src/k8s.io/kubernetes/_output/local/bin/linux/amd64/ hyperkube"

rm -f hyades-hyperkube_1.6.4-1_amd64.deb
fpm --vendor 'MIT SIPB Hyades Project' $FPMOPT
cp hyades-hyperkube_1.6.4-1_amd64.deb ../binaries
