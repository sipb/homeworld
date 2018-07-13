#!/bin/bash
set -e -u
cd "$(dirname "$0")"

BRANCH="${1:-master}"

if [ -e "upstream" ]
then
	(cd upstream && git checkout "$BRANCH" && git pull)
else
	git clone https://github.com/sipb/homeworld-upstream.git upstream -b "$BRANCH"
fi
cd upstream
sha512sum --check ../SHA512SUM.UPSTREAM || (echo "CHECKSUM MISMATCH" && false)
COUNT_UPSTREAM=$(find -type f | grep -vF README.md | grep -vE '^\./snapshot\.debian\.org/archive/debian/[0-9]+T[0-9]+Z/pool/' | grep -vE '^[.]/[.]git/' | wc -l)
COUNT_HERE=$(wc -l <../SHA512SUM.UPSTREAM)
if [ "${COUNT_UPSTREAM}" != "${COUNT_HERE}" ]
then
	echo "FILES COUNT DIFFERED: ${COUNT_HERE} EXPECTED BUT ${COUNT_UPSTREAM} FOUND"
	exit 1
fi
