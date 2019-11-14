#!/bin/bash
set -e -u

cd "$(dirname "$0")"

if [ "${HOMEWORLD_DEPLOY_CHROOT:-}" = "" ] || [ ! -e "$(dirname HOMEWORLD_DEPLOY_CHROOT)" ]
then
    echo "invalid path to chroot: ${HOMEWORLD_DEPLOY_CHROOT:-}" 1>&2
    echo '(have you populated $HOMEWORLD_DEPLOY_CHROOT?)'
    echo '(try export HOMEWORLD_DEPLOY_CHROOT=$HOME/chroot)'
    exit 1
fi

if [[ "${HOMEWORLD_DEPLOY_CHROOT}" = *" "* ]]
then
    echo "chroot name cannot include a space" 1>&2
    exit 1
fi

if [ -e "${HOMEWORLD_DEPLOY_CHROOT}" ]
then
    echo "chroot already exists" 1>&2
    exit 1
fi

if [ "$USER" = "root" ]
then
    echo "this script should be run as a user with sudo capabilities" 1>&2
    exit 1
fi

mkdir -m 'u=rwx,go=rx' "${HOMEWORLD_DEPLOY_CHROOT}"

sudo debootstrap --include="$(grep -vE '^#' packages.list | tr '[:space:]' '\n' | sed '/^$/d' | tr '\n' ,)" stretch "${HOMEWORLD_DEPLOY_CHROOT}" http://debian.csail.mit.edu/debian/
sudo chroot "${HOMEWORLD_DEPLOY_CHROOT}" apt-get update
sudo chroot "${HOMEWORLD_DEPLOY_CHROOT}" groupadd "$(id -gn)" -g "$(id -g)"
sudo chroot "${HOMEWORLD_DEPLOY_CHROOT}" useradd -m -u "$(id -u)" -g "$(id -g)" -G kvm "$USER" -s "/bin/bash"
sudo bash -c "echo '$USER ALL=(ALL) NOPASSWD:ALL' >>'${HOMEWORLD_DEPLOY_CHROOT}/etc/sudoers'"
sudo bash -c "cat >>${HOMEWORLD_DEPLOY_CHROOT}/etc/bash.bashrc" <<EOF
export PS1="\[\033[01;31m\][hyades deploy] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF
cat >>"${HOMEWORLD_DEPLOY_CHROOT}/home/$USER/.bashrc" <<EOF
export PS1="\[\033[01;31m\][hyades deploy] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF
mkdir -m 'u=rwx,go=' "${HOMEWORLD_DEPLOY_CHROOT}/home/$USER/.ssh"
ssh-keygen -t rsa -N "" -f "${HOMEWORLD_DEPLOY_CHROOT}/home/$USER/.ssh/id_rsa"
echo "127.0.0.1 $(basename "$HOMEWORLD_DEPLOY_CHROOT")" | sudo bash -c "cat >>'$HOMEWORLD_DEPLOY_CHROOT/etc/hosts'"

echo "Done!"
