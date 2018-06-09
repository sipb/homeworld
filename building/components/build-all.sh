#!/bin/bash
set -e -u

for x in homeworld-*/
do
	(cd "$x" && ./build-package.sh)
done

./clean.sh

echo "Built all!"
