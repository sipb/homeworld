#!/usr/bin/env python3
"""
update-defs: change a set of Bazel variable declarations in a particular file.

For example, if a file named `debian-iso/deps.bzl` had the following lines in the middle:

    VERSION = '20190702+deb10u2'
    RELEASE = 'buster'
    MINI_ISO_HASH = 'fa713fb9ab7853de6aefe6b83c58bcdba19ef79a0310de84d40dd2754a9539d7'

And you ran the following command with the following input:

    $ ./update-defs.py debian-iso/deps.bzl
    VERSION = '20190702+deb10u3'
    RELEASE = 'buster'
    MINI_ISO_HASH = '26b6f6f6bcb24c4e59b965d4a2a6c44af5d79381b9230d69a7d4db415ddcb4cd'

Then those three lines (and only those lines) would be updated in `deps.bzl`.
"""

import re
import sys


def is_valid_identifier(text):
    return bool(re.match("^[A-Z0-9_]+$", text))


def get_match(line):
    if "=" in line:
        match = line.split("=",1)[0].strip()
        if is_valid_identifier(match):
            return match


def parse_changes(changes):
    mapping = {}
    for line in changes.split("\n"):
        if not line: continue
        match = get_match(line)
        if match is None:
            raise ValueError("could not parse change line: %s" % repr(line))
        if match in mapping:
            raise KeyError("duplicate input mapping: %s" % match)
        mapping[match] = line
    return mapping


def apply_changes(line, lookup, used):
    match = get_match(line)
    if match is not None and match in lookup:
        used[match] += 1
        line = lookup[match]
        print("=>", line)
    return line


def perform_changes(filename, changes):
    with open(filename, "r") as f:
        lines = f.read().split("\n")
    lookup = parse_changes(changes)
    used = {var: 0 for var in lookup}
    lines = [apply_changes(line, lookup, used) for line in lines]
    for var, usedcount in used.items():
        if usedcount < 1:
            raise ValueError("did not use requested change: %s was not used" % (repr(var)))
        elif usedcount > 1:
            raise ValueError("used requested change multiple times: %s was used %d times" % (repr(var), usedcount))
    with open(filename, "w") as f:
        f.write("\n".join(lines))
    print()
    print("updated", len(lookup), "entries")


def main(argv):
    if len(argv) != 2:
        print("usage: %s <target file>", file=sys.stderr)
        sys.exit(1)
    changes = sys.stdin.read()
    perform_changes(sys.argv[1], changes)


if __name__ == "__main__":
    main(sys.argv)
