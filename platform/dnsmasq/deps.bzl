load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def dnsmasq_dependencies():
    http_archive(
        name = "dnsmasq",
        urls = ["http://www.thekelleys.org.uk/dnsmasq/dnsmasq-2.78.tar.xz"],
        sha256 = "89949f438c74b0c7543f06689c319484bd126cc4b1f8c745c742ab397681252b",
        build_file = "//dnsmasq:BUILD.import",
    )
