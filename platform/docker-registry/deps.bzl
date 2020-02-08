load("@bazel_gazelle//:deps.bzl", "go_repository")

def docker_registry_dependencies():
    go_repository(
        name = "com_github_docker_distribution",
        commit = "2461543d988979529609e8cb6fca9ca190dc48da",  # 2.7.1
        importpath = "github.com/docker/distribution",
        build_external = "vendored",
        patches = ["//docker-registry:docker-registry.patch"],
    )
