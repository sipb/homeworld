load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def knc_dependencies():
    http_archive(
        name = "knc",
        urls = ["http://oskt.secure-endpoints.com/downloads/knc-1.7.1.tar.gz"],
        sha256 = "0e24873f2a5228e1814749becbecd67d643bb9f63663cf85a7aeedf8e73de40f",
        build_file = "//knc:BUILD.import",
    )
