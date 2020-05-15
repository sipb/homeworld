load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

def go_dependencies():
    go_rules_dependencies()

    # TODO: stop using binaries built by upstream; use our own
    go_register_toolchains(
        # TODO: might need to bump kubernetes to support this newer version
        go_version = "1.13.10",
    )
    protobuf_deps()
    gazelle_dependencies()
