#!/bin/bash
set -e -u

if [ "${HOMEWORLD_DEPLOY_CHROOT:-}" = "" ] || [ ! -e "${HOMEWORLD_DEPLOY_CHROOT}" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_DEPLOY_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_DEPLOY_CHROOT?)'
    echo '(have you created a chroot?)'
    exit 1
fi

cd "$(dirname "$0")"
HERE="$(pwd)"
cd "$(git rev-parse --show-toplevel)"
REL="$(realpath --relative-to "$(pwd)" "$HERE")"
sudo systemd-nspawn \
     -E PATH="/usr/local/bin:/usr/bin:/bin:/homeworld/deploy-chroot" \
     -E HOMEWORLD_DIR="/cluster" \
     -E HOMEWORLD_DISASTER="/disaster" \
     --register=no \
     -M "$(basename "$HOMEWORLD_DEPLOY_CHROOT")" \
     --bind "$(pwd)":/homeworld:norbind \
     --bind "$HOMEWORLD_DIR":/cluster:norbind \
     --bind "$HOMEWORLD_DISASTER":/disaster:norbind \
     -u "$USER" \
     -a \
     -D "$HOMEWORLD_DEPLOY_CHROOT" \
     --capability=CAP_NET_ADMIN \
     "/homeworld/$REL/enter-internal.sh"
