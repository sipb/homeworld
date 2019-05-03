#!/bin/bash
set -e -u

sudo apt-get -y purge homeworld-apt-setup && sudo apt-get -y remove 'homeworld-*' && sudo apt-get clean || true
sudo dpkg -i /homeworld/apt-setup.deb
sudo apt-get update
sudo apt-get -y install homeworld-spire
