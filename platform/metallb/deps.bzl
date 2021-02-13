load("@bazel_gazelle//:deps.bzl", "go_repository")

def metallb_dependencies():
    go_repository(
        name = "tf_universe_go_metallb",
        commit = "15d9ed5cce53b457a955f894aa58e7f4856f73d5",  # v0.8.3
        importpath = "go.universe.tf/metallb",
    )

    go_repository(
        name = "com_github_go_kit_kit",
        commit = "150a65a7ec6156b4b640c1fd55f26fd3d475d656",  # v0.9.0
        importpath = "github.com/go-kit/kit",
    )

    go_repository(
        name = "com_github_go_logfmt_logfmt",
        commit = "390ab7935ee28ec6b286364bba9b4dd6410cb3d5",  # v0.3.0
        importpath = "github.com/go-logfmt/logfmt",
    )

    go_repository(
        name = "com_github_golang_groupcache",
        commit = "02826c3e79038b59d737d3b1c0a1d937f71a4433",
        importpath = "github.com/golang/groupcache",
    )

    go_repository(
        name = "com_github_hashicorp_golang_lru",
        commit = "20f1fb78b0740ba8c3cb143a61e86ba5c8669768",  # v0.5.0
        importpath = "github.com/hashicorp/golang-lru",
    )

    go_repository(
        name = "com_github_mdlayher_arp",
        commit = "98a83c8a27177c5179d02d41ad50b0cce8e59338",
        importpath = "github.com/mdlayher/arp",
    )

    go_repository(
        name = "com_github_mdlayher_ethernet",
        commit = "0394541c37b7f86a10e0b49492f6d4f605c34163",
        importpath = "github.com/mdlayher/ethernet",
    )

    go_repository(
        name = "com_github_mdlayher_ndp",
        commit = "012988d57f9ae7e329f16ec2a86b37c771cb57e8",
        importpath = "github.com/mdlayher/ndp",
    )

    go_repository(
        name = "com_github_mdlayher_raw",
        commit = "fef19f00fc18511f735e13972bc53266d5a53f8c",
        importpath = "github.com/mdlayher/raw",
    )

    go_repository(
        name = "com_github_mikioh_ipaddr",
        commit = "d465c8ab672111787b24b8f03326449059a4aa33",
        importpath = "github.com/mikioh/ipaddr",
    )

    go_repository(
        name = "com_gitlab_golang_commonmark_puny",
        commit = "2cd490539afe7c6fc0eda6c59ef88fa93a00ea0d",
        importpath = "gitlab.com/golang-commonmark/puny",
    )
