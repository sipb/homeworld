#!/bin/bash
set -e -u

# interacts with preseed code

if grep -q TEMPORARY-KEYCLIENT-CONFIGURATION /etc/homeworld/config/keyclient.yaml
then
    source /etc/homeworld/config/local.conf
    cp /etc/homeworld/config/keyclient-${KIND}.yaml /etc/homeworld/config/keyclient.yaml
    systemctl restart keyclient.service
    hostnamectl set-hostname "${HOST_NODE}"
    update-ca-certificates
fi
