#!/bin/bash
set -e -u

cd "$(dirname "$0")"
bazel build //apt-setup:package.deb
cp "$(bazel info bazel-bin)"/apt-setup/package.deb "$(git rev-parse --show-toplevel)"/apt-setup.deb
chmod 'u=rw,go=r' "$(git rev-parse --show-toplevel)"/apt-setup.deb
