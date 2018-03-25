#!/bin/bash
set -e -u

cd "$(dirname "$0")"

if [ "${HOMEWORLD_CHROOT:-}" = "" -o ! -e "$(dirname HOMEWORLD_CHROOT)" ]
then
	echo "invalid path to chroot: ${HOMEWORLD_CHROOT:-}" 1>&2
	echo '(have you populated $HOMEWORLD_CHROOT?)'
	echo '(try export HOMEWORLD_CHROOT=$HOME/chroot)'
	exit 1
fi

if [ -e "${HOMEWORLD_CHROOT}" ]
then
	echo "chroot already exists" 1>&2
	exit 1
fi

mkdir "${HOMEWORLD_CHROOT}"
sudo debootstrap --include="$(cat chroot-packages.list | tr '\n' ',' | sed 's/,$//')" stretch "${HOMEWORLD_CHROOT}" http://debian.csail.mit.edu/debian/
ln -sT "homeworld/building" "${HOMEWORLD_CHROOT}/h"
sudo chroot "${HOMEWORLD_CHROOT}" useradd -m -u "$UID" "$USER" -s "/bin/bash"

echo "Done!"
