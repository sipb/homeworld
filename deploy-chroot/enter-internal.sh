#!/bin/bash

set -euo pipefail

sudo mount -o rw,remount /proc/sys
if grep -wq kvm /proc/misc; then
    sudo mknod --mode='ug=rw,o=' /dev/kvm c 10 "$(grep -w kvm /proc/misc | cut -f 1 -d ' ')"
    sudo chgrp kvm /dev/kvm
else
    echo 'note: no kvm support; not creating /dev/kvm' >&2
fi
cd /cluster
exec ssh-agent bash
