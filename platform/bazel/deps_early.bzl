load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def bazel_dependencies_early():
    git_repository(
        name = "io_bazel_rules_docker",
        remote = "https://github.com/bazelbuild/rules_docker",
        commit = "5647f4f7a6b7c247e788675963e2e03a6e7156e1",
    )
    http_archive(
        name = "rules_pkg",
        url = "https://github.com/bazelbuild/rules_pkg/releases/download/0.1.0/rules_pkg-0.1.0.tar.gz",
        sha256 = "752146e2813f4c135ec9f71b592bf98f96f026049e6d65248534dbeccb2448e1",
    )
