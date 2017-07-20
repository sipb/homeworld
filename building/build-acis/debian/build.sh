#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"
EXTRA_PACKAGES="wget,curl,ca-certificates,git,realpath,file,less,gnupg,python,python3,iptables,iputils-ping,iputils-arping,iproute2,bzip2,gzip,net-tools,netcat-traditional,dnsutils"
DEBVER=20170719T213259Z
UPDATE_HASH=bae2a744fe760be3e3b9195806e9890054ba01a585780a0975da8bacbcc6eaa2

debian_bootstrap

clean_apt_files
clean_ld_aux
clean_pycache

write_debian_image
