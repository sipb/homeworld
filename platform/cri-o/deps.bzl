load("@bazel_gazelle//:deps.bzl", "go_repository")

def cri_o_dependencies():
    go_repository(
        name = "com_github_cri_o_cri_o",
        commit = "dd73a465144f71031728f0de8439ddda08c98119",  # 1.16.3
        importpath = "github.com/cri-o/cri-o",
        build_external = "vendored",
        build_file_proto_mode = "disable_global",
        patches = [
            # most of this is only required because the #cgo pkg-config directive is not correctly processed by Gazelle
            "//cri-o:build.patch",
        ],
        patch_args = ["-p1"],
    )

    go_repository(
        name = "com_github_containers_conmon",
        commit = "1bddbf7051a973f4a4fecf06faa0c48e82f1e9e1",  # 2.0.15
        importpath = "github.com/containers/conmon",
        build_file_generation = "off",
        patches = [
            "//cri-o:conmon.patch",
        ],
        patch_args = ["-p1"],
    )
