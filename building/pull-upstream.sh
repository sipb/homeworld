#!/bin/bash
set -e -u
cd "$(dirname "$0")"

COMMIT="$(cat upstream.commit)"

if [ ! -e "upstream" ]
then
	git clone https://github.com/sipb/homeworld-upstream.git upstream -b master
fi
cd upstream
git fetch
git checkout -B working "${COMMIT}"

sha512sum --check ../SHA512SUM.UPSTREAM || (echo "CHECKSUM MISMATCH" && false)
COUNT_UPSTREAM=$(find -type f | grep -vF README.md | grep -vE '^\./snapshot\.debian\.org/archive/debian/[0-9]+T[0-9]+Z/pool/' | grep -vE '^[.]/[.]git/' | wc -l)
COUNT_HERE=$(wc -l <../SHA512SUM.UPSTREAM)
if [ "${COUNT_UPSTREAM}" != "${COUNT_HERE}" ]
then
	echo "FILES COUNT DIFFERED: ${COUNT_HERE} EXPECTED BUT ${COUNT_UPSTREAM} FOUND"
	exit 1
fi
