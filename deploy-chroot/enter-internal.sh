#!/bin/bash

set -euo pipefail

sudo mount -o rw,remount /proc/sys
sudo chgrp kvm /dev/kvm
cd /cluster
exec ssh-agent bash
