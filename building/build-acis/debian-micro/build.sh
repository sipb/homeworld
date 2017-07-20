#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"
DEBVER=20170719T213259Z
UPDATE_HASH=0cabcba72e46bd0168158e18f9aa7384aa06931bdb3b06e41de8c0e822c2ee33

debian_bootstrap

force_remove_packages e2fslibs e2fsprogs login
force_remove_packages apt bash base-files base-passwd debian-archive-keyring gpgv init-system-helpers tzdata sysvinit-utils mount adduser
force_remove_packages --force-depends perl-base debconf
force_remove_packages --force-depends dpkg
clean_apt_files
clean_ld_aux
clean_doc_files
clean_locales

write_debian_image
