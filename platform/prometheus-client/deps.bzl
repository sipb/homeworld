load("@bazel_gazelle//:deps.bzl", "go_repository")

def prometheus_client_dependencies():
    # prometheus client, for packages like pull-monitor

    go_repository(
        name = "com_github_prometheus_client_golang",
        commit = "aa9238db679fc02bf22cf4f2c27a980edcb5ada0", # v1.5.1
        importpath = "github.com/prometheus/client_golang",
    )

    go_repository(
        name = "com_github_prometheus_common",
        commit = "d978bcb1309602d68bb4ba69cf3f8ed900e07308", # v0.9.1
        importpath = "github.com/prometheus/common",
    )

    go_repository(
        name = "com_github_prometheus_procfs",
        commit = "46159f73e74d1cb8dc223deef9b2d049286f46b1", # v0.0.11
        importpath = "github.com/prometheus/procfs",
    )

    go_repository(
        name = "com_github_prometheus_client_model",
        commit = "7bc5445566f0fe75b15de23e6b93886e982d7bf9", # v0.2.0
        importpath = "github.com/prometheus/client_model",
    )

    go_repository(
        name = "com_github_cespare_xxhash_v2",
        commit = "d7df74196a9e781ede915320c11c378c1b2f3a1f", # v2.1.1
        importpath = "github.com/cespare/xxhash/v2",
        remote = "https://github.com/cespare/xxhash.git",
        vcs = "git",
    )

    go_repository(
        name = "com_github_matttproud_golang_protobuf_extensions",
        commit = "c12348ce28de40eed0136aa2b644d0ee0650e56c", # v1.0.1
        importpath = "github.com/matttproud/golang_protobuf_extensions",
    )

    go_repository(
        name = "com_github_beorn7_perks",
        commit = "37c8de3658fcb183f997c4e13e8337516ab753e6", # v1.0.1
        importpath = "github.com/beorn7/perks",
    )
