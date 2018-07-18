#!/bin/bash
set -e -u

if [ "${HOMEWORLD_CHROOT:-}" = "" -o ! -e "${HOMEWORLD_CHROOT}" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_CHROOT?)'
    echo '(have you created a chroot?)'
    exit 1
fi

ORIG_PWD="$(pwd)"
cd "$(dirname "$0")"
ORIG_REL="$(realpath --relative-to "$(pwd)/building" "${ORIG_PWD}")"
if [[ "${ORIG_REL}" =~ \.\. ]]
then
	ORIG_REL=""
fi
if [ -e "$HOME/.gnupg/pubring.kbx" ]
then
	mkdir -p "$HOMEWORLD_CHROOT/home/$USER/.gnupg/private-keys-v1.d/"
	chmod 0700 "$HOMEWORLD_CHROOT/home/$USER/.gnupg"
	cp "$HOME/.gnupg/pubring.kbx" "$HOMEWORLD_CHROOT/home/$USER/.gnupg/pubring.kbx"
	cp "$HOME/.gnupg/trustdb.gpg" "$HOMEWORLD_CHROOT/home/$USER/.gnupg/trustdb.gpg"
	cp -R "$HOME/.gnupg/private-keys-v1.d/"* "$HOMEWORLD_CHROOT/home/$USER/.gnupg/private-keys-v1.d"
fi
sudo systemd-nspawn -M "$(basename $HOMEWORLD_CHROOT)" --bind $(pwd):/homeworld:norbind -u "$USER" -a -D "$HOMEWORLD_CHROOT" bash -c "cd /h/${ORIG_REL} && gpg-agent --daemon --keep-tty && sudo nginx && exec bash"
