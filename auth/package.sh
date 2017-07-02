#!/bin/bash
set -e -u

VERSION=0.1.4

cd $(dirname $0)
HERE=$(pwd)
HYBIN=$(pwd)/../binaries/
mkdir -p ${HYBIN}

FPMOPT="-s dir -t deb"
FPMOPT="$FPMOPT -n hyades-authserver -v ${VERSION} --iteration 1"
FPMOPT="$FPMOPT --maintainer 'sipb-hyades-root@mit.edu'"
FPMOPT="$FPMOPT --license MIT -a x86_64"
FPMOPT="$FPMOPT --depends krb5-user"
FPMOPT="$FPMOPT --after-install auth.postinstall --after-remove auth.postremove --before-install auth.preinstall"
FPMOPT="$FPMOPT hyauth=/usr/bin/hyauth sshd_config=/etc/ssh/sshd_config.hyades-new"

fpm --vendor 'MIT SIPB Hyades Project' $FPMOPT

echo "authserver package built!"
