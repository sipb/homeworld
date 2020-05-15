load("@bazel_gazelle//:deps.bzl", "go_repository")

def etcd_dependencies():
    # WARNING: etcd is pulled in twice! once here, and once as a dependency for Rook!
    # These are not necessarily the same version, and the other one doesn't use vendored deps!
    go_repository(
        name = "etcd",
        commit = "d57e8b8d97adfc4a6c224fe116714bf1a1f3beb9",  # 3.3.12
        importpath = "github.com/coreos/etcd",
        build_external = "vendored",
        build_file_proto_mode = "disable_global",
        prepatch_cmds = [
            # to get etcd's vendoring strategy to be compatible with gazelle
            "cp -RT cmd/vendor vendor",
            "rm -r cmd/vendor",
        ],
        patches = ["//etcd:etcd-visibility.patch", "//etcd:etcdctl-visibility.patch"],
    )
