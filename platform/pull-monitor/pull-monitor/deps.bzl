load("@bazel_gazelle//:deps.bzl", "go_repository")

# TODO: figure out if this is needed for anything else
def prometheus_client_dependencies():
    # prometheus client, for packages like pull-monitor

    go_repository(
        name = "com_github_prometheus_client_golang",
        commit = "967789050ba94deca04a5e84cce8ad472ce313c1",
        importpath = "github.com/prometheus/client_golang",
    )

    go_repository(
        name = "com_github_prometheus_common",
        commit = "b36ad289a3eaecdc52470a19591146a2c0ffb532",
        importpath = "github.com/prometheus/common",
    )

    go_repository(
        name = "com_github_prometheus_procfs",
        commit = "abf152e5f3e97f2fafac028d2cc06c1feb87ffa5",
        importpath = "github.com/prometheus/procfs",
    )

    go_repository(
        name = "com_github_prometheus_client_model",
        commit = "5c3871d89910bfb32f5fcab2aa4b9ec68e65a99f",
        importpath = "github.com/prometheus/client_model",
    )

    go_repository(
        name = "com_github_matttproud_golang_protobuf_extensions",
        commit = "fc2b8d3a73c4867e51861bbdd5ae3c1f0869dd6a",
        importpath = "github.com/matttproud/golang_protobuf_extensions",
    )

    go_repository(
        name = "com_github_beorn7_perks",
        commit = "3ac7bf7a47d159a033b107610db8a1b6575507a4",
        importpath = "github.com/beorn7/perks",
    )

def pull_monitor_dependencies():
    prometheus_client_dependencies()

    go_repository(
        name = "com_github_hashicorp_errwrap",
        commit = "8a6fb523712970c966eefc6b39ed2c5e74880354",  # 1.0.0
        importpath = "github.com/hashicorp/errwrap",
    )

    go_repository(
        name = "com_github_hashicorp_go_multierror",
        commit = "886a7fbe3eb1c874d46f623bfa70af45f425b3d1",  # 1.0.0
        importpath = "github.com/hashicorp/go-multierror",
    )

    go_repository(
        name = "com_github_pkg_errors",
        commit = "27936f6d90f9c8e1145f11ed52ffffbfdb9e0af7",
        importpath = "github.com/pkg/errors",
    )
