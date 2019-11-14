#!/bin/bash

set -eu

cd "$(dirname "$0")"

packages=()
mapfile -t packages < <(grep -vE '^#' packages.list | tr '[:space:]' '\n' | sed '/^$/d')
sudo apt-get install -y "${packages[@]}"
