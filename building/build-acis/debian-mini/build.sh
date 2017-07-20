#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"

debian_bootstrap

force_remove_packages e2fslibs e2fsprogs login
clean_apt_files
clean_doc_files
clean_locales

write_debian_image
