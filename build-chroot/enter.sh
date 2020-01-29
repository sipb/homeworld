#!/bin/bash
set -e -u

if [ "${HOMEWORLD_CHROOT:-}" = "" ] || [ ! -e "${HOMEWORLD_CHROOT}" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_CHROOT?)'
    echo '(have you created a chroot?)'
    exit 1
fi

ORIG_PWD="$(pwd)"
cd "$(dirname "$0")"
cd "$(git rev-parse --show-toplevel)"
ORIG_REL="$(realpath --relative-to "$(pwd)" "${ORIG_PWD}")"
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
if [ -e "$HOME/.ssh" ]
then
	mkdir -p "$HOMEWORLD_CHROOT/home/$USER/.ssh/"
	cp -R "$HOME/.ssh/." "$HOMEWORLD_CHROOT/home/$USER/.ssh"
fi
if [ ! -e platform/upload/version-cache ]
then
	echo "{}" >platform/upload/version-cache
fi
if [ "${HOMEWORLD_UPLOAD_BIND:-}" != "" ]
then
	UPLOAD_ARGS=("--bind" "${HOMEWORLD_UPLOAD_BIND}:${HOMEWORLD_UPLOAD_BIND}:norbind")
fi

# gpg --list-keys is needed here for the situation where the host for the
# chroot has an older version of GPG, and it needs to upgrade the database
# before it can be used. if we don't do this, we experience strange issues with
# bazel run //upload not being able to find a key that definitely exists.

sudo systemd-nspawn -E PATH="/usr/local/bin:/usr/bin:/bin" --register=no -M "$(basename "$HOMEWORLD_CHROOT")" --bind "$(pwd)":/homeworld:norbind "${UPLOAD_ARGS[@]}" -u "$USER" -a -D "$HOMEWORLD_CHROOT" bash -c "cd /homeworld/${ORIG_REL} && gpg-agent --daemon --keep-tty && gpg --list-keys >/dev/null && exec bash"
