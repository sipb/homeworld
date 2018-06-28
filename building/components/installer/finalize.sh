#!/bin/bash
set -e -u
# usage: commit.sh <output-file>
xorriso -as mkisofs -o "$1" -b boot/isolinux.bin -c boot/boot.cat -no-emul-boot -boot-load-size 4 -boot-info-table .
