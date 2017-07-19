#!/bin/bash
set -e -u

(cd helper-go && ./build.sh)
(cd helper-acbuild && ./build.sh)

for x in homeworld-*/
do
	(cd $x && ./build-package.sh)
done

./clean.sh

echo "Built all!"
