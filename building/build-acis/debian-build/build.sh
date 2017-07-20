#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"
EXTRA_PACKAGES="wget,curl,ca-certificates,git,realpath,file,less,gnupg,python,python3,bzip2,gzip,make,gcc,binutils,automake,autoconf,libc6-dev,cpio,squashfs-tools,xz-utils,patch,bc,libacl1-dev,libssl-dev,libsystemd-dev,zlib1g-dev,pkg-config,libglib2.0-dev,libpixman-1-dev,libcap-dev"
DEBVER=20170719T213259Z
UPDATE_HASH=e649ee484e556f41b50e77da060ef354d3c292ed8f32a34c76da50a909ce24ae

debian_bootstrap

clean_apt_files
clean_ld_aux
clean_pycache

write_debian_image
