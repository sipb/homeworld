#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/debian.sh

RELEASE="stretch"
EXTRA_PACKAGES="wget,curl,ca-certificates,git,realpath,file,less,gnupg,python,python3,iptables,iputils-ping,iputils-arping,iproute2,bzip2,gzip,net-tools,netcat-traditional,dnsutils"
DEBVER=20171105T160402Z

debian_bootstrap

clean_apt_files
clean_ld_aux
clean_pycache
clean_resolv_conf

write_debian_image
