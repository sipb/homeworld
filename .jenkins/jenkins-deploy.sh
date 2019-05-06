#!/bin/bash

set -eu

# WARNING: This script runs outside of Docker, so there is no isolation
# from the rest of the system. This includes Jenkins itself.
# So resist the temptation to apt-get upgrade or autoremove.

sudo rm -rf /var/www/html/apt/autobuild
sudo cp -r /var/homeworld-binaries/autobuild /var/www/html/apt/autobuild
sudo apt-get -qq purge -y 'homeworld-*' || true
sudo apt-get clean
# should match only one binary
sudo dpkg -i /var/homeworld-binaries/autobuild/pool/main/h/homeworld-apt-setup/homeworld-apt-setup_*.deb
sudo apt-get -qq update
sudo apt-get -qq install -y homeworld-spire
find /var/homeworld-deploy/ -mindepth 1 -delete
cp .jenkins/deploy-setup.yaml /var/homeworld-deploy/setup.yaml

export HOMEWORLD_DISASTER="/var/homeworld-deploy/disaster.key"
export HOMEWORLD_DIR="/var/homeworld-deploy"

pwgen -s 160 1 > "$HOMEWORLD_DISASTER"
spire authority gen

rm -rf "$HOME/.ssh/"
mkdir -p "$HOME/.ssh/"
ssh-keygen -t rsa -N "" -f "$HOME/.ssh/id_rsa"
eval "$(ssh-agent -s)"
(cd /var/homeworld-deploy && (spire virt net down || true))
(cd /var/homeworld-deploy && spire virt auto install)
