#!/bin/bash
set -e -u

VERSION="0.28.1"

cd "$(dirname "$0")"
curl -L -o "bazel-${VERSION}.deb" "https://github.com/bazelbuild/bazel/releases/download/${VERSION}/bazel_${VERSION}-linux-x86_64.deb"
sha256sum --check "bazel-${VERSION}.deb.sha256"
sudo dpkg "$@" --install "bazel-${VERSION}.deb"
rm "bazel-${VERSION}.deb"
