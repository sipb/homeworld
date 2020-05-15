load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

VERSION = "20190702+deb10u4"
RELEASE = "buster"
MINI_ISO_HASH = "a749cb499cf4f335d1d4fbffb041353f6e41f5d0681b1d856cb167ed2c27e756"

def debian_iso_dependencies():
    http_file(
        name = "debian_iso",
        urls = [
            "http://ftp.debian.org/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
            "http://debian.csail.mit.edu/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
        ],
        sha256 = MINI_ISO_HASH,
    )
