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

if [[ "${HOMEWORLD_CHROOT}" =~ " " ]]
then
    echo "chroot name cannot include a space" 1>&2
    exit 1
fi

if [ -e "${HOMEWORLD_CHROOT}" ]
then
    echo "chroot already exists" 1>&2
    exit 1
fi

if [ "$USER" = "root" ]
then
    echo "this script should be run as a user with sudo capabilities" 1>&2
    exit 1
fi

mkdir -m 'u=rwx,go=rx' "${HOMEWORLD_CHROOT}"
sudo debootstrap --include="$(cat packages.list | grep -vE '^#' | tr '\n ' ',,' | sed 's/,$//' | sed 's/,,/,/g')" stretch "${HOMEWORLD_CHROOT}" http://debian.csail.mit.edu/debian/
# TODO: build our own Bazel, rather than grabbing it from Google's repo
echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee "${HOMEWORLD_CHROOT}/etc/apt/sources.list.d/bazel.list"
cat bazel-release.pub.gpg | sudo chroot "${HOMEWORLD_CHROOT}" apt-key add -
sudo chroot "${HOMEWORLD_CHROOT}" apt-get update
sudo chroot "${HOMEWORLD_CHROOT}" apt-get install -y bazel
sudo chroot "${HOMEWORLD_CHROOT}" groupadd "$(id -gn)" -g "$(id -g)"
sudo chroot "${HOMEWORLD_CHROOT}" useradd -m -u "$(id -u)" -g "$(id -g)" "$USER" -s "/bin/bash"
sudo chroot "${HOMEWORLD_CHROOT}" pip install -U gsutil pyasn1
sudo bash -c "echo '$USER ALL=(ALL) NOPASSWD:ALL' >>'${HOMEWORLD_CHROOT}/etc/sudoers'"
sudo bash -c "cat >>${HOMEWORLD_CHROOT}/etc/bash.bashrc" <<EOF
export PS1="\[\033[01;31m\][homeworld] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF
sudo bash -c "cat >>${HOMEWORLD_CHROOT}/home/$USER/.bashrc" <<EOF
export PS1="\[\033[01;31m\][homeworld] \[\033[01;32m\]\u\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\]\$ "
EOF
echo "127.0.0.1 $(basename "$HOMEWORLD_CHROOT")" | sudo bash -c "cat >>'$HOMEWORLD_CHROOT/etc/hosts'"

echo "Done!"
