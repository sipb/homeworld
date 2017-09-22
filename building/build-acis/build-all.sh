#!/bin/bash
set -e -u

for x in debian debian-build debian-micro debian-mini flannel
do
	(cd "$x" && ./build.sh)
done

echo "Built all containers!"
