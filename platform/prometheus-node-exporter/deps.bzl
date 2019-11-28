load("@bazel_gazelle//:deps.bzl", "go_repository")

def prometheus_node_exporter_dependencies():
    go_repository(
        name = "com_github_prometheus_node_exporter",
        commit = "98bc64930d34878b84a0f87dfe6e1a6da61e532d",
        importpath = "github.com/prometheus/node_exporter",
        build_external = "vendored",
        patches = ["//prometheus-node-exporter:visibility.patch"],
    )
