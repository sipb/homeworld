#!/bin/bash
set -e -u

if [ "${HOMEWORLD_DEPLOY_CHROOT:-}" = "" -o ! -e "${HOMEWORLD_DEPLOY_CHROOT}" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_DEPLOY_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_DEPLOY_CHROOT?)'
    echo '(have you created a chroot?)'
    exit 1
fi

cd "$(dirname "$0")"
cd "$(git rev-parse --show-toplevel)"
sudo systemd-nspawn -E PATH="/usr/local/bin:/usr/bin:/bin" -M "$(basename $HOMEWORLD_DEPLOY_CHROOT)" --bind $(pwd):/homeworld:norbind -u "$USER" -a -D "$HOMEWORLD_DEPLOY_CHROOT" bash -c "cd /homeworld && (sudo apt-get -y purge homeworld-apt-setup && sudo apt-get -y remove 'homeworld-*' && sudo apt-get clean || true) && sudo dpkg -i apt-setup.deb && sudo apt-get update && sudo apt-get -y install homeworld-spire"
