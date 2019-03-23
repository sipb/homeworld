#!/bin/bash
set -e -u

# interacts with preseed code

if [ "$(cat /etc/homeworld/config/keyserver.variant)" = "base" ]
then
    source /etc/homeworld/config/local.conf
    systemctl restart keyclient.service
    hostnamectl set-hostname "${HOST_NODE}"
    update-ca-certificates
fi
