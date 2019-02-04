#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf

exec /usr/bin/aci-pull-monitor "homeworld.private/pullcheck:${HOMEWORLD_VERSION}"
