#! /usr/bin/env bash
set -e

# TODO: Migrate to nix flakes once promoted from experimental to stable

# Modified from https://gist.github.com/gdamjan/8158b57379932fd0e07ce6d83399b71f
nix-channel --add https://nixos.org/channels/nixos-22.05 nixpkgs
nix-channel --list
nix-channel --update


# Install nixos-generators: https://github.com/nix-community/nixos-generators
nix-env -f https://github.com/nix-community/nixos-generators/archive/master.tar.gz -i

nixos-generate --configuration /homeworld/hosts/qemu_vm/configuration.nix  --system x86_64-linux --format qcow --out-link workdir
cp workdir/nixos.qcow2 /homeworld/hosts/qemu_vm
