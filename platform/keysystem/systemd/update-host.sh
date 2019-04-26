#!/bin/bash
set -e -u

source /etc/homeworld/config/local.conf
hostnamectl set-hostname "${HOST_NODE}"
update-ca-certificates
