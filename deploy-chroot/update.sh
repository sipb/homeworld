#!/bin/bash

set -eu

cd "$(dirname "$0")"
sudo apt-get install -y $(cat packages.list | grep -vE '^#' | tr '\n,' '  ')
