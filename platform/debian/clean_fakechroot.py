#!/usr/bin/env python3
"""cleans up any symbolic links pointing with absolute paths to the build directory itself"""
import os
import sys

if len(sys.argv) != 2:
    print("usage:", sys.argv[0], "<ROOTFS>", file=sys.stderr)
    sys.exit(1)

rootfs = os.path.abspath(sys.argv[1])

for root, dirs, files in os.walk(rootfs):
    for f in files:
        path = os.path.join(root, f)
        if not os.path.islink(path):
            continue
        full_link = os.readlink(path)
        if not os.path.isabs(full_link):
            continue
        rootrel = os.path.relpath(full_link, rootfs)
        if rootrel.split("/")[0] == "..":
            # doesn't point within the rootfs; nothing to do
            continue
        os.remove(path)
        os.symlink(os.path.join("/", rootrel), path)

# We have a Jenkins-only bug where /proc is a symlink to itself in the flannel
# container, which breaks a lot of things. We don't know why, exactly, this is
# happening -- but we need to mitigate it.
if os.path.islink(os.path.join(rootfs, "proc")):
    os.unlink(os.path.join(rootfs, "proc"))
    os.mkdir(os.path.join(rootfs, "proc"))
