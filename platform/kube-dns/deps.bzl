load("//bazel:gorepo_patchfix.bzl", "go_repository_alt")

def kube_dns_dependencies():
    go_repository_alt(
        name = "com_github_kubernetes_dns",
        commit = "b8d50e0e7698317816e7e6b27d48f0988098e6fc", # 1.14.13
        importpath = "k8s.io/dns",
        build_external = "vendored",
        prepatch_cmds = ["find vendor/k8s.io/kubernetes -name BUILD -delete"],
        postpatches = ["//kube-dns:dns-visibility.patch"],
    )
