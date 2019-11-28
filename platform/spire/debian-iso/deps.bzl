load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

def debian_iso_dependencies():
    http_file(
        name = "debian_iso",
        urls = [
            "http://ftp.debian.org/debian/dists/stretch/main/installer-amd64/20170615+deb9u7+b2/images/netboot/mini.iso",
            "http://debian.csail.mit.edu/debian/dists/stretch/main/installer-amd64/20170615+deb9u7+b2/images/netboot/mini.iso",
        ],
        sha256 = "6a1f4457430285dcd6882829eb5b96e840e06b0e792c2bdf2d91ce552da19d84",
    )
