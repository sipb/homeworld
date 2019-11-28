load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")

def bazel_dependencies():
    container_repositories()
