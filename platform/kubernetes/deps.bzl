load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_gazelle//:deps.bzl", "go_repository")

# kubernetes client application (like flannel-monitor) dependencies
def kubernetes_client_dependencies():
    go_repository(
        name = "com_github_gregjones_httpcache",
        commit = "787624de3eb7bd915c329cba748687a3b22666a6",
        importpath = "github.com/gregjones/httpcache",
    )

    go_repository(
        name = "com_github_peterbourgon_diskv",
        commit = "5f041e8faa004a95c88a202771f4cc3e991971e6",
        importpath = "github.com/peterbourgon/diskv",
    )

def kubernetes_dependencies():
    git_repository(
        name = "io_k8s_repo_infra",
        remote = "https://github.com/kubernetes/repo-infra/",
        commit = "9f4571ad7242bf3ec4b47365062498c2528f9a5f",
    )

    http_archive(
        name = "kubernetes",
        sha256 = "3f430156abcee1930f1eb0e7bd853c0b411e33f8a43e5b52207c0a49d58eb85c",
        type = "tar.gz",
        urls = ["https://dl.k8s.io/v1.16.0/kubernetes-src.tar.gz"],
        patches = ["//kubernetes:0001-fix-bazel-compat.patch"],
    )
