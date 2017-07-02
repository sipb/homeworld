#!/bin/bash
set -e -u
# git clone git@github.com:sipb/homeworld-upstream.git
if [ -e "upstream" ]
then
	(cd upstream && git pull)
else
	git clone https://github.com/sipb/homeworld-upstream.git upstream
fi
cd upstream
sha512sum --check ../SHA512SUM.UPSTREAM || (echo "CHECKSUM MISMATCH" && false)
COUNT_UPSTREAM=$(find -type f | grep -vF README.md | grep -vE '^[.]/[.]git/' | wc -l)
COUNT_HERE=$(wc -l <../SHA512SUM.UPSTREAM)
if [ "${COUNT_UPSTREAM}" != "${COUNT_HERE}" ]
then
	echo "FILES COUNT DIFFERED: ${COUNT_HERE} EXPECTED BUT ${COUNT_UPSTREAM} FOUND"
	exit 1
fi
