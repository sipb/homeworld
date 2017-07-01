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
sha512sum --check ../SHA512SUM.UPSTREAM
