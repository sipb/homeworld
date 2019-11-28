load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def go_dependencies_early():
    # TODO: audit all downloads, make sure they're all source code
    http_archive(
        name = "io_bazel_rules_go",
        urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.19.5/rules_go-v0.19.5.tar.gz"],
        sha256 = "513c12397db1bc9aa46dd62f02dd94b49a9b5d17444d49b5a04c5a89f3053c1c",
    )

    http_archive(
        name = "bazel_gazelle",
        urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
        sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    )

    git_repository(
        name = "com_google_protobuf",
        commit = "6a59a2ad1f61d9696092f79b6d74368b4d7970a3", # 3.9.0
        remote = "https://github.com/protocolbuffers/protobuf",
        shallow_since = "1558721209 -0700",
    )
