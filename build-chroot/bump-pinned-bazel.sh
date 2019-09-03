#!/bin/bash
set -e -u

# This script bumps the version of bazel to a newer version.

BASEPATH="$(realpath "$(dirname "$0")")"

WORKDIR=$(mktemp -d)
trap "rm -r $WORKDIR" EXIT

cd "$WORKDIR"

if [ "${1:-}" = "" ]
then
	echo "usage: bump-pinned-bazel.sh <version>" 1>&2
	exit 1
fi

VERSION="$1"

curl -L -o "bazel-${VERSION}.deb" "https://github.com/bazelbuild/bazel/releases/download/${VERSION}/bazel_${VERSION}-linux-x86_64.deb"
curl -L -o "bazel-${VERSION}.deb.sig" "https://github.com/bazelbuild/bazel/releases/download/${VERSION}/bazel_${VERSION}-linux-x86_64.deb.sig"
gpg --no-default-keyring --keyring "${BASEPATH}/bazel-keyring.gpg" --trust-mode=always --verify "bazel-${VERSION}.deb.sig"
sha256sum "bazel-${VERSION}.deb" >"${BASEPATH}/bazel-${VERSION}.deb.sha256"
rm "bazel-${VERSION}.deb" "bazel-${VERSION}.deb.sig"
sed -i 's/^VERSION=".*"$/VERSION="'"${VERSION}"'"/' "${BASEPATH}/install-bazel.sh"

echo "Bazel bumped. Please take the following steps:"
echo "  - git add bazel-${VERSION}.deb.sha256 install-bazel.sh"
echo "  - Remove the old bazel .sha256 entry"
echo "  - Make a commit with the first line:"
echo "      bazel: bump to ${VERSION}"
