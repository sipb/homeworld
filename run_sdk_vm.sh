#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

# Prereqs
# - KVM Virtualuzation available (scripts/kvm_ok)
# - Docker

# TODO: Remove dependency from docker/docker.io
#docker run -it --rm \
#            --device /dev/kvm \
#            -v "$(pwd)":/homeworld \
#            docker.io/nixos/nix \
#            /bin/sh /homeworld/scripts/setup_vm.sh

set -o xtrace
qemu-system-x86_64 -enable-kvm -snapshot -nographic \
    -cpu host \
    -m 4096 \
    -nic user,model=virtio,hostfwd=tcp::2222-:22 \
    -drive if=virtio,file=hosts/qemu_vm/nixos.qcow2 \
    -virtfs local,path="$(pwd)",mount_tag=host0,security_model=none,id=host0,readonly \
    -monitor unix:/tmp/qemu-monitor-nixos.socket,server,nowait
