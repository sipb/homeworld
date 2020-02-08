load("@bazel_gazelle//:deps.bzl", "go_repository")

def cri_tools_dependencies():
    go_repository(
        name = "com_github_kubernetes_sigs_cri_tools",
        commit = "b7d3a78a3587c400136aac8fc5e6727cb3b3e67a",  # v1.14.0
        importpath = "github.com/kubernetes-sigs/cri-tools",
        build_file_proto_mode = "disable_global",
    )
