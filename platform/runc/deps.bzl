load("@bazel_gazelle//:deps.bzl", "go_repository")

def runc_dependencies():
    go_repository(
        name = "com_github_opencontainers_runc",
        commit = "029124da7af7360afa781a0234d1b083550f797c", # v1.0.0-rc7 plus a few patches for a regression
        importpath = "github.com/opencontainers/runc",
    )
