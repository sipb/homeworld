load("@bazel_gazelle//:deps.bzl", "go_repository")

def kube_dns_dependencies():
    go_repository(
        name = "com_github_kubernetes_dns",
        commit = "b8d50e0e7698317816e7e6b27d48f0988098e6fc",  # 1.14.13
        importpath = "k8s.io/dns",
        build_external = "vendored",
        prepatch_cmds = ["find vendor/k8s.io/kubernetes -name BUILD -delete"],
        patches = ["//kube-dns:dns-visibility.patch"],
    )
