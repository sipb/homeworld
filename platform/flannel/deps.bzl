load("@bazel_gazelle//:deps.bzl", "go_repository")

def flannel_dependencies():
    go_repository(
        name = "com_github_coreos_flannel",
        commit = "d3eea7f5cdb895965394eb5f34645cdc3b535d5b", # 0.11.0
        importpath = "github.com/coreos/flannel",
    )
