load("@bazel_gazelle//:deps.bzl", "go_repository")

def keysystem_dependencies():
    go_repository(
        name = "org_golang_x_sys",
        commit = "fae7ac547cb717d141c433a2a173315e216b64c4",
        importpath = "golang.org/x/sys",
    )

    go_repository(
        name = "org_golang_x_crypto",
        commit = "88737f569e3a9c7ab309cdc09a07fe7fc87233c3",
        importpath = "golang.org/x/crypto",
    )

    go_repository(
        name = "in_gopkg_yaml_v2",
        commit = "51d6538a90f86fe93ac480b35f37b2be17fef232", # 2.2.2
        importpath = "gopkg.in/yaml.v2",
    )
