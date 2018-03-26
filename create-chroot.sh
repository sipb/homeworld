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
sudo ln -sT "homeworld/building" "${HOMEWORLD_CHROOT}/h"
sudo chroot "${HOMEWORLD_CHROOT}" useradd -m -u "$UID" "$USER" -s "/bin/bash"
sudo bash -c "echo '$USER ALL=(ALL) NOPASSWD:ALL' >>'${HOMEWORLD_CHROOT}/etc/sudoers'"
sudo bash -c "cat >>${HOMEWORLD_CHROOT}/etc/bash.bashrc" <<EOF
export PS1="\[\033[01;31m\][homeworld] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF
sudo bash -c "cat >>${HOMEWORLD_CHROOT}/home/$USER/.bashrc" <<EOF
export PS1="\[\033[01;31m\][homeworld] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF

echo "Done!"
