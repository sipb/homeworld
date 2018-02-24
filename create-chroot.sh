#!/bin/bash
set -e -u

cd "$(dirname "$0")"

if [ "$HOMEWORLD_CHROOT" = "" -o ! -e "$(dirname HOMEWORLD_CHROOT)" ]
then
	echo "invalid path to chroot: $HOMEWORLD_CHROOT" 1>&2
	echo '(have you populated $HOMEWORLD_CHROOT?)'
	exit 1
fi

if [ -e "${CHROOT}" ]
then
	echo "chroot already exists" 1>&2
	exit 1
fi

mkdir "${CHROOT}"
sudo debootstrap --include="$(cat chroot-packages.list | tr '\n' ',' | sed 's/,$//')" stretch "${CHROOT}" http://debian.csail.mit.edu/debian/

echo "Done!"
