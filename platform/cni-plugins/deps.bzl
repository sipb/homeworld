load("@bazel_gazelle//:deps.bzl", "go_repository")

def cni_plugins_dependencies():
    go_repository(
        name = "com_github_containernetworking_plugins",
        commit = "a62711a5da7a2dc2eb93eac47e103738ad923fd6", # 0.7.5
        importpath = "github.com/containernetworking/plugins",
    )
