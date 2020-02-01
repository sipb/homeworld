load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

def bazel_dependencies_early():
    git_repository(
        name = "io_bazel_rules_docker",
        remote = "https://github.com/bazelbuild/rules_docker",
        commit = "5647f4f7a6b7c247e788675963e2e03a6e7156e1",
    )
