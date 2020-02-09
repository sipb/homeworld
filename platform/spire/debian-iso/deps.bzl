load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

VERSION = "20190702+deb10u3"
RELEASE = "buster"
MINI_ISO_HASH = "26b6f6f6bcb24c4e59b965d4a2a6c44af5d79381b9230d69a7d4db415ddcb4cd"

def debian_iso_dependencies():
    http_file(
        name = "debian_iso",
        urls = [
            "http://ftp.debian.org/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
            "http://debian.csail.mit.edu/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
        ],
        sha256 = MINI_ISO_HASH,
    )
