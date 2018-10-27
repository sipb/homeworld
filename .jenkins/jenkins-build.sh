#!/bin/bash

set -eu

apt-get -qq install -y git gnupg sudo psmisc
apt-get -qq install -y build-essential cpio squashfs-tools debootstrap realpath
# use same uid/gid as host so that the jenkins user has permissions to
# work in the repository
JENKINS_UID=$(stat -c '%u' .)
JENKINS_GID=$(stat -c '%g' .)
groupadd --gid $JENKINS_GID jenkins
adduser --uid $JENKINS_UID --gid $JENKINS_GID --disabled-password --gecos "" jenkins
echo 'jenkins  ALL=(ALL:ALL) NOPASSWD:ALL' >> /etc/sudoers
# use su instead of su - to keep the pwd
su jenkins -c 'building/pull-upstream.sh'
su jenkins -c 'HOMEWORLD_CHROOT="$HOME/autobuild-chroot" ./create-chroot.sh'
echo 'make -C upstream-check/ -j2 verify && (killall gpg-agent || true)' | su jenkins -c 'HOMEWORLD_CHROOT="$HOME/autobuild-chroot" ./enter-chroot-ci.sh'
su jenkins -c 'gpg --batch --gen-key .jenkins/gpg-key-details'
su jenkins -c 'python .jenkins/generate-branches-config.py'
echo "mkdir -p binaries && glass components -b jenkins-ci --upload" | su jenkins -c 'HOMEWORLD_CHROOT="$HOME/autobuild-chroot" ./enter-chroot-ci.sh'
rm -rf /binaries/autobuild
mv building/binaries /binaries/autobuild
