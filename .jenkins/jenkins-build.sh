#!/bin/bash

set -eu

while true
do
	sleep 1
done

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
su jenkins -c 'HOMEWORLD_CHROOT="$HOME/autobuild-chroot" ./build-chroot/create.sh'
su jenkins -c 'gpg --batch --gen-key .jenkins/gpg-key-details'
su jenkins -c 'python .jenkins/generate-branches-config.py'
echo "bazel run //upload --verbose_failures" | su jenkins -c 'HOMEWORLD_CHROOT="$HOME/autobuild-chroot" ./build-chroot/enter-ci.sh'
rm -rf /binaries/autobuild
mv binaries /binaries/autobuild
