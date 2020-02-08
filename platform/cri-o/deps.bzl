load("@bazel_gazelle//:deps.bzl", "go_repository")

def cri_o_dependencies():
    go_repository(
        name = "com_github_cri_o_cri_o",
        commit = "b7316701c17ebc7901d10a716f15e66008c52525",  # 1.15.2
        importpath = "github.com/cri-o/cri-o",
        build_external = "vendored",
        build_file_proto_mode = "disable_global",
        patches = [
            # most of this is only required because the #cgo pkg-config directive is not correctly processed by Gazelle
            "//cri-o:build.patch",
        ],
        patch_args = ["-p1"],
    )
