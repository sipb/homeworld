#!/bin/bash
set -e -u

cd "$(git rev-parse --show-toplevel)"
pushd platform
bazel build //apt-setup:package.deb
output="$(bazel info bazel-bin)/apt-setup/package.deb"
popd
cp "$output" apt-setup.deb
chmod 'u=rw,go=r' apt-setup.deb
