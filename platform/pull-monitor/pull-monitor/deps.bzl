load("@bazel_gazelle//:deps.bzl", "go_repository")

def pull_monitor_dependencies():
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
