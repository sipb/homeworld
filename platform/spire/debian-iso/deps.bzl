load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

VERSION = '20190702+deb10u2'
RELEASE = 'buster'
MINI_ISO_HASH = 'fa713fb9ab7853de6aefe6b83c58bcdba19ef79a0310de84d40dd2754a9539d7'

def debian_iso_dependencies():
    http_file(
        name = "debian_iso",
        urls = [
            "http://ftp.debian.org/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
            "http://debian.csail.mit.edu/debian/dists/" + RELEASE + "/main/installer-amd64/" + VERSION + "/images/netboot/mini.iso",
        ],
        sha256 = MINI_ISO_HASH,
    )
