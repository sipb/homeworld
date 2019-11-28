load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

def bazel_dependencies_early():
    git_repository(
        name = "io_bazel_rules_docker",
        remote = "https://github.com/bazelbuild/rules_docker",
        commit = "968e0b7c8b3bc7e009531231ac926325cd2745bc",
    )
