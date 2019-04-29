#!/bin/bash

set -eu

cd "$(dirname "$0")"
sudo apt-get install -y $(cat chroot-packages.list | grep -vE '^#' | tr '\n,' '  ')
