#!/usr/bin/env python3
import subprocess
import apt_pkg
import sys
import os
apt_pkg.init_system()

if len(sys.argv) != 2 or sys.argv[1][0:1] != "2" or sys.argv[1][-1] != "Z":
	print("usage: snapshot-pull.py <snapshot-version>")
	print("   version example: 20180710T043017Z")
	sys.exit(1)

VERSION = sys.argv[1]
prefix = "snapshot.debian.org/archive/debian/%s" % VERSION

def download(path):
	assert not os.path.isabs(path)
	assert ".." not in path.split("/")
	path = os.path.join(prefix, path)
	print("downloading", path)
	parent_dir = os.path.dirname(path)
	if not os.path.isdir(parent_dir):
		os.makedirs(parent_dir)
	subprocess.check_call(["curl", "-s", "-L", "http://" + path, "-o", path])

for path in ["./dists/stretch/Release.gpg", "./dists/stretch/Release"]:
	download(path)

for arch in ["amd64", "all"]:
	for path in ["Release", "Packages.xz", "Packages.gz"]:
		download("./dists/stretch/main/binary-%s/%s" % (arch, path))

packages = [line.split(":")[0] for line in sys.stdin.read().split()]

def parse_packages_file(path):
	each_package = subprocess.check_output(["zcat", "--", os.path.join(prefix, path)]).decode().strip().split("\n\n")
	each_package = [parse_desc(package) for package in each_package]
	mapping = {}
	for package in each_package:
		name = package["Package"]
		if name in mapping and apt_pkg.version_compare(package["Version"], mapping[name]["Version"]) < 0:
			continue
		mapping[name] = package
	return mapping

def parse_desc(text):
	return dict(x.split(": ", 1) for x in text.strip().replace("\n ", " ").split("\n"))

mapping = {}

for arch in ["amd64", "all"]:
	mapping.update(parse_packages_file("./dists/stretch/main/binary-%s/Packages.gz" % arch))

for package in packages:
	download(mapping[package]["Filename"])
