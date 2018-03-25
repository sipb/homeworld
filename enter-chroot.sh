#!/bin/bash
set -e -u

if [ "${HOMEWORLD_CHROOT:-}" = "" -o ! -e "${HOMEWORLD_CHROOT}" ]
then
        echo "invalid path to chroot: ${HOMEWORLD_CHROOT:-}" 1>&2
        echo '(have you populated $HOMEWORLD_CHROOT?)'
	echo '(have you created a chroot?)'
        exit 1
fi

MOUNTDIR="${HOMEWORLD_CHROOT}/homeworld/"

if [ -e "${MOUNTDIR}" ]
then
	sudo umount "${MOUNTDIR}" || true
	rmdir "${MOUNTDIR}"
fi
sudo umount "${HOMEWORLD_CHROOT}/proc" || true
mkdir "${MOUNTDIR}"
sudo mount --bind "$(dirname "$0")" "${MOUNTDIR}"
sudo mount -t proc proc "${HOMEWORLD_CHROOT}/proc"

cat >"${HOMEWORLD_CHROOT}/_enter.sh" <<EOF
#!/bin/bash
cd /h/
export PS1="\[\033[01;31m\][homeworld] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
history -c
export HOME="/home/$USER"
export HISTFILE="$HOME/.bash_history"
history -r
alias ls='ls --color=auto'
rm /_enter.sh
EOF
chmod +x "${HOMEWORLD_CHROOT}/_enter.sh"

sudo chroot --user="$USER" "${HOMEWORLD_CHROOT}" bash --rcfile /_enter.sh || true
sudo umount "${HOMEWORLD_CHROOT}/proc"
sudo umount "${MOUNTDIR}"
rmdir "${MOUNTDIR}"
