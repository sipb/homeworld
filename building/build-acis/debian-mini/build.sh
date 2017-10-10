#!/bin/bash
set -e -u
cd "$(dirname "$0")"
source ../common/debian.sh

RELEASE="stretch"
DEBVER=20171004T154711Z

debian_bootstrap

force_remove_packages e2fslibs e2fsprogs login
clean_apt_files
clean_ld_aux
clean_doc_files
clean_locales
clean_resolv_conf

write_debian_image
