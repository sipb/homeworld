#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/debian.sh

RELEASE="stretch"
DEBVER=20171004T154711Z

debian_bootstrap

force_remove_packages e2fslibs e2fsprogs login
force_remove_packages apt bash base-files base-passwd debian-archive-keyring gpgv init-system-helpers tzdata sysvinit-utils mount adduser
force_remove_packages --force-depends perl-base debconf
force_remove_packages --force-depends dpkg
clean_apt_files
clean_ld_aux
clean_doc_files
clean_locales
clean_resolv_conf

write_debian_image
