#!/bin/bash
set -e -u
cd "$(dirname $0)"
source ../common/debian.sh

RELEASE="stretch"
DEBVER=20170719T213259Z
UPDATE_HASH=4dad4cd6591bd00715f2cc5d4dab2ae675f4f38e9d675084e3680b1176660a17

debian_bootstrap

force_remove_packages e2fslibs e2fsprogs login
clean_apt_files
clean_ld_aux
clean_doc_files
clean_locales

write_debian_image
