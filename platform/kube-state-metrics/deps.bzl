load("@bazel_gazelle//:deps.bzl", "go_repository")

def kube_state_metrics_dependencies():
    go_repository(
        name = "com_github_kubernetes_kube_state_metrics",
        commit = "4c0e83b3407e489eda34c26f7794ec69856ccd76",  # v1.7.2
        importpath = "k8s.io/kube-state-metrics",
    )
