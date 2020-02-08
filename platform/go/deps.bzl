load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

def go_dependencies():
    go_rules_dependencies()

    # TODO: stop using binaries built by upstream; use our own
    go_register_toolchains(
        go_version = "1.12.10",
    )
    protobuf_deps()
    gazelle_dependencies()
