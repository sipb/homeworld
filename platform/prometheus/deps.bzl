load("@bazel_gazelle//:deps.bzl", "go_repository")

def prometheus_dependencies():
    go_repository(
        name = "com_github_prometheus_prometheus",
        commit = "0a74f98628a0463dddc90528220c94de5032d1a0",
        importpath = "github.com/prometheus/prometheus",
        build_external = "vendored",
        build_file_proto_mode = "disable_global",
        patches = ["//prometheus:prometheus-visibility.patch"],
    )
