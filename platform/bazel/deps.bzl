load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")
load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")
load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")
load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

def bazel_dependencies():
    http_archive(
        name = "containerregistry",
        sha256 = "a8cdf2452323e0fefa4edb01c08b2ec438c9fa3192bc9f408b89287598c12abc",
        strip_prefix = "containerregistry-0.0.36",
        urls = [("https://github.com/google/containerregistry/archive/v0.0.36.tar.gz")],
        patches = ["//bazel:0001-containerregistry-py2.patch"],
    )

    container_repositories()
    rules_pkg_dependencies()
    protobuf_deps()
    rules_proto_dependencies()
    rules_proto_toolchains()
    buildifier_dependencies()
