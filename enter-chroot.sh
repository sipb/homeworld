#!/bin/bash
set -e -u

if [ "${HOMEWORLD_CHROOT:-}" = "" -o ! -e "${HOMEWORLD_CHROOT}" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_CHROOT?)'
    echo '(have you created a chroot?)'
    exit 1
fi

sudo systemd-nspawn -M homeworld --bind $(pwd):/homeworld:norbind -u "$USER" -a -D "$HOMEWORLD_CHROOT" bash -c "cd /h/ && exec bash"
