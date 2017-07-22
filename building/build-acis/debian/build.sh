#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"
EXTRA_PACKAGES="wget,curl,ca-certificates,git,realpath,file,less,gnupg,python,python3,iptables,iputils-ping,iputils-arping,iproute2,bzip2,gzip,net-tools,netcat-traditional,dnsutils"
DEBVER=20170719T213259Z
UPDATE_HASH=c00ba4408e03bb388b2e176afb8e4882ea0d073fef4ae0adda02d479f56d2610

debian_bootstrap

clean_apt_files
clean_ld_aux
clean_pycache

write_debian_image
