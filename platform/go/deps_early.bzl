load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def go_dependencies_early():
    # TODO: audit all downloads, make sure they're all source code
    http_archive(
        name = "io_bazel_rules_go",
        urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.22.3/rules_go-v0.22.3.tar.gz"],
        sha256 = "db2b2d35293f405430f553bc7a865a8749a8ef60c30287e90d2b278c32771afe",
    )

    http_archive(
        name = "bazel_gazelle",
        urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz"],
        sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
    )

    git_repository(
        name = "com_google_protobuf",
        commit = "6a59a2ad1f61d9696092f79b6d74368b4d7970a3",  # 3.9.0
        remote = "https://github.com/protocolbuffers/protobuf",
        shallow_since = "1558721209 -0700",
    )
