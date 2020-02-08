load("@bazel_gazelle//:deps.bzl", "go_repository")

def oci_tools_dependencies():
    go_repository(
        name = "com_github_containers_skopeo",
        commit = "404c5bd341ccb383061f4eb505f24d2801b31b94",  # v0.1.35
        importpath = "github.com/containers/skopeo",
        patches = ["//oci-tools:skopeo.patch"],
        patch_args = ["-p1"],
    )

    go_repository(
        name = "com_github_opencontainers_image_tools",
        commit = "7f6433100c1757a65c72f374080b6899f8152075",  # v1.0.0-rc1
        importpath = "github.com/opencontainers/image-tools",
        patches = ["//oci-tools:image-tools.patch"],
    )
