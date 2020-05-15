load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_gazelle//:deps.bzl", "go_repository")

# kubernetes client application (like flannel-monitor) dependencies
def kubernetes_client_dependencies():
    go_repository(
        name = "io_k8s_apimachinery",
        commit = "6a84e37a896db9780c75367af8d2ed2bb944022e",  # 1.14.1
        importpath = "k8s.io/apimachinery",
        build_file_proto_mode = "disable_global",
    )

    go_repository(
        name = "io_k8s_client_go",
        commit = "1a26190bd76a9017e289958b9fba936430aa3704",  # 1.14.1
        importpath = "k8s.io/client-go",
        build_file_proto_mode = "disable_global",
    )

    go_repository(
        name = "io_k8s_api",
        commit = "6e4e0e4f393bf5e8bbff570acd13217aa5a770cd",  # 1.14.1
        importpath = "k8s.io/api",
        build_file_proto_mode = "disable_global",
    )

    go_repository(
        name = "com_github_imdario_mergo",
        commit = "9316a62528ac99aaecb4e47eadd6dc8aa6533d58",
        importpath = "github.com/imdario/mergo",
    )

    go_repository(
        name = "com_github_google_gofuzz",
        commit = "24818f796faf91cd76ec7bddd72458fbced7a6c1",
        importpath = "github.com/google/gofuzz",
    )

    go_repository(
        name = "io_k8s_kube_openapi",
        commit = "b3a7cee44a305be0a69e1b9ac03018307287e1b0",
        importpath = "k8s.io/kube-openapi",
    )

    go_repository(
        name = "com_github_googleapis_gnostic",
        commit = "0c5108395e2debce0d731cf0287ddf7242066aba",
        importpath = "github.com/googleapis/gnostic",
        build_file_proto_mode = "disable_global",
    )

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

    go_repository(
        name = "com_github_json_iterator_go",
        commit = "ab8a2e0c74be9d3be70b3184d9acc634935ded82",
        importpath = "github.com/json-iterator/go",
    )

    go_repository(
        name = "com_github_google_btree",
        commit = "7d79101e329e5a3adf994758c578dab82b90c017",
        importpath = "github.com/google/btree",
    )

    go_repository(
        name = "in_gopkg_inf_v0",
        commit = "3887ee99ecf07df5b447e9b00d9c0b2adaa9f3e4",
        importpath = "gopkg.in/inf.v0",
    )

    go_repository(
        name = "com_github_spf13_pflag",
        commit = "583c0c0531f06d5278b7d917446061adc344b5cd",
        importpath = "github.com/spf13/pflag",
    )

    go_repository(
        name = "org_golang_x_time",
        commit = "f51c12702a4d776e4c1fa9b0fabab841babae631",
        importpath = "golang.org/x/time",
    )

    go_repository(
        name = "com_github_modern_go_reflect2",
        commit = "94122c33edd36123c84d5368cfb2b69df93a0ec8",
        importpath = "github.com/modern-go/reflect2",
    )

    go_repository(
        name = "com_github_modern_go_concurrent",
        commit = "bacd9c7ef1dd9b15be4a9909b8ac7a4e313eec94",
        importpath = "github.com/modern-go/concurrent",
    )

    go_repository(
        name = "org_golang_x_net",
        commit = "da137c7871d730100384dbcf36e6f8fa493aef5b",
        importpath = "golang.org/x/net",
    )

    go_repository(
        name = "org_golang_x_oauth2",
        commit = "a6bd8cefa1811bd24b86f8902872e4e8225f74c4",
        importpath = "golang.org/x/oauth2",
    )

    go_repository(
        name = "org_golang_x_text",
        commit = "f21a4dfb5e38f5895301dc265a8def02365cc3d0",  # 0.3.0
        importpath = "golang.org/x/text",
    )

    go_repository(
        name = "io_k8s_klog",
        commit = "8e90cee79f823779174776412c13478955131846",
        importpath = "k8s.io/klog",
    )

    go_repository(
        name = "io_k8s_sigs_yaml",
        commit = "fd68e9863619f6ec2fdd8625fe1f02e7c877e480",
        importpath = "sigs.k8s.io/yaml",
    )

    go_repository(
        name = "io_k8s_utils",
        commit = "c2654d5206da6b7b6ace12841e8f359bb89b443c",
        importpath = "k8s.io/utils",
    )

    go_repository(
        name = "com_github_davecgh_go_spew",
        commit = "782f4967f2dc4564575ca782fe2d04090b5faca8",
        importpath = "github.com/davecgh/go-spew",
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
