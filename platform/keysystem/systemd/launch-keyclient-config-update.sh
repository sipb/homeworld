#!/bin/bash
set -e -u

# interacts with preseed code

if grep -q base /etc/homeworld/config/keyserver.variant
then
    source /etc/homeworld/config/local.conf
    echo "${KIND}" >/etc/homeworld/config/keyserver.variant
    keyconfgen
    systemctl restart keyclient.service
    hostnamectl set-hostname "${HOST_NODE}"
    update-ca-certificates
fi
