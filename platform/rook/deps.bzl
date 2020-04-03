load("@bazel_gazelle//:deps.bzl", "go_repository")
load("//go:helper.bzl", "importpath_to_repo_name")
load("//rook/tini:deps.bzl", "tini_dependencies")

# WARNING: etcd is pulled in twice! once to build the etcd binaries, and once here!
# These are not necessarily the same version, and the other one uses vendored dependencies!
# The same applies to kubernetes!

# these versions are extracted from the Rook go.sum file
ROOK_DEPS = """
github.com/aws/aws-sdk-go v1.16.26 h1:GWkl3rkRO/JGRTWoLLIqwf7AWC4/W/1hMOUZqmX0js4=
github.com/banzaicloud/k8s-objectmatcher v1.1.0 h1:KHWn9Oxh21xsaGKBHWElkaRrr4ypCDyrh15OB1zHtAw=
github.com/blang/semver v3.5.1+incompatible h1:cQNTCjp13qL8KC3Nbxr/y2Bqb63oX6wdnnjpJbkM4JQ=
github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e h1:Wf6HqHfScWJN9/ZjdUKyjop4mf3Qdd+1TvvltAvM3m8=
github.com/coreos/pkg v0.0.0-20180108230652-97fdf19511ea h1:n2Ltr3SrfQlf/9nOna1DoGKxLx3qTSI8Ttl6Xrqp6mw=
github.com/coreos/prometheus-operator v0.34.0 h1:TF9qaydNeUamLKs0hniaapa4FBz8U8TIlRRtJX987A4=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/emicklei/go-restful v2.10.0+incompatible h1:l6Soi8WCOOVAeCo4W98iBFC6Og7/X8bpRt51oNLZ2C8=
github.com/evanphx/json-patch v4.5.0+incompatible h1:ouOWdg56aJriqS0huScTkVXPC5IcNrDCXZ6OoTAWu7M=
github.com/fsnotify/fsnotify v1.4.7 h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=
github.com/ghodss/yaml v1.0.0 h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=
github.com/go-ini/ini v1.51.1 h1:/QG3cj23k5V8mOl4JnNzUNhc1kr/jzMiNsNuWKcx8gM=
github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 h1:uHTyIjqVhYRhLbJ8nIiOJHkEZZ+5YoOsAbD3sk82NiE=
github.com/go-logr/logr v0.1.0 h1:M1Tv3VzNlEHg6uyACnRdtrploV2P7wZqH8BoQMtz0cg=
github.com/googleapis/gnostic v0.3.1 h1:WeAefnSUHlBb0iJKwxFDZdbfGwkd7xRNuV+IpXMJhYk=
github.com/google/btree v1.0.0 h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=
github.com/google/go-cmp v0.3.1 h1:Xye71clBPdm5HgqGwUkwhbynsUJZhDbS20FvLhQ2izg=
github.com/google/gofuzz v1.0.0 h1:A8PeW59pxE9IoFRqBp37U+mSNaQoZ46F1f0f863XSXw=
github.com/google/uuid v1.1.1 h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=
github.com/go-openapi/jsonpointer v0.19.3 h1:gihV7YNZK1iK6Tgwwsxo2rJbD1GTbdm72325Bq8FI3w=
github.com/go-openapi/jsonreference v0.19.3 h1:5cxNfTy0UVC3X8JL5ymxzyoUZmo8iZb+jeTWn7tUa8o=
github.com/go-openapi/spec v0.19.3 h1:0XRyw8kguri6Yw4SxhsQA/atC88yqrk0+G4YhI2wabc=
github.com/go-openapi/swag v0.19.5 h1:lTz6Ys4CmqqCQmZPBlbQENR1/GucA2bzYTE12Pw4tFY=
github.com/goph/emperror v0.17.2 h1:yLapQcmEsO0ipe9p5TaN22djm3OFV/TfM/fcYP0/J18=
github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 h1:Ovs26xHkKqVztRpIrF/92BcuyuQ/YW4NSIpoGtfXNho=
github.com/hashicorp/golang-lru v0.5.3 h1:YPkqC67at8FYaadspW/6uE0COsBxS2656RLEr8Bppgk=
github.com/hashicorp/go-version v1.2.0 h1:3vNe/fWF5CBgRIguda1meWhsZHy3m8gCJ5wx+dIzX/E=
github.com/imdario/mergo v0.3.7 h1:Y+UAYTZ7gDEuOfhxKWy+dvb5dRQ6rJjFSdX2HZY1/gI=
github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af h1:pmfjZENx5imkbgOkpRUYLnmbU7UEFbjtDA2hxJ1ichM=
github.com/json-iterator/go v1.1.8 h1:QiWkFLKq0T7mpzwOTu6BzNDbfTE8OLrYhVKYMLF46Ok=
github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20200401090632-ee27f62faef8 h1:/+OIW+inkJRBJlIHQqMUXRbYHmQLZwj7lqIV/TUecsE=
github.com/kube-object-storage/lib-bucket-provisioner v0.0.0-20200107223247-51020689f1fb h1:0tLk5jekrSwmQJqoyHW6oVAdfG7DbLAaobKcURwCRoY=
github.com/mailru/easyjson v0.7.0 h1:aizVhC/NAAcKWb+5QsU1iNOZb4Yws5UO2I+aIprQITM=
github.com/miekg/dns v1.1.4 h1:rCMZsU2ScVSYcAsOXgmC6+AKOK+6pmQTOcw03nfwYV0=
github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=
github.com/modern-go/reflect2 v1.0.1 h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=
github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 h1:C3w9PqII01/Oq1c1nUAm88MOHcQC9l5mIlSMApZMrHA=
github.com/NYTimes/gziphandler v0.0.0-20170623195520-56545f4a5d46 h1:lsxEuwrXEAokXB9qhlbKWPpo3KMLZQ5WB5WLQRW1uq0=
github.com/openshift/cluster-api v0.0.0-20191129101638-b09907ac6668 h1:IDZyg/Kye98ptqpc9j9rzPjZJlijjEDe8g7TZ67CmLU=
github.com/openshift/machine-api-operator v0.2.1-0.20190903202259-474e14e4965a h1:mcl6pEpG0ZKeMnAMhtmcoy7jFY8PcMRHmxdRQmowxo4=
github.com/pborman/uuid v1.2.0 h1:J7Q5mO4ysT1dv8hyrUGHb9+ooztCXu1D8MY8DZYsu3g=
github.com/PuerkitoBio/purell v1.1.1 h1:WEQqlqaGbrPkxLJWfBwQmfEAE1Z7ONdDLqrN38tNFfI=
github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 h1:d+Bc7a5rLufV/sSk/8dngufqelfh6jnri85riMAaF/M=
github.com/spf13/afero v1.2.2 h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=
github.com/spf13/cobra v0.0.5 h1:f0B+LkLX6DtmRH1isoNA9VTtNUK9K8xYd28JNNfOv/s=
github.com/spf13/pflag v1.0.5 h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=
github.com/yanniszark/go-nodetool v0.0.0-20191206125106-cd8f91fa16be h1:e8XjnroTyruokenelQLRje3D3nbti3ol45daXg5iWUA=
go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738 h1:VcrIfasaLFkyjk6KNlXQSzO+B0fZcnECiDrKJsfxka0=
golang.org/x/net v0.0.0-20200226121028-0de0cce0169b h1:0mm1VjtFUOIlE1SbDlwjYaDxZVDP2S5ou6y0gSgXHu8=
golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 h1:SVwTIAaPC2U/AvvLNZ2a7OVsmBpC8L5BlwK1whH3hm0=
golang.org/x/text v0.3.2 h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=
golang.org/x/time v0.0.0-20191024005414-555d28b269f0 h1:/5xXl8Y5W96D+TtHSlonuFqGHIWVuyCkGJLwGh9JJFs=
gomodules.xyz/jsonpatch/v2 v2.0.1 h1:xyiBuvkD2g5n7cYzx6u2sxQvsAy4QJsZFCzGVdzOXZ0=
google.golang.org/grpc v1.23.1 h1:q4XQuHFC6I28BKZpo6IYyb3mNO+l7lSOxRuYTCiDfXk=
gopkg.in/fsnotify.v1 v1.4.7 h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=
gopkg.in/inf.v0 v0.9.1 h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=
gopkg.in/square/go-jose.v2 v2.2.2 h1:orlkJ3myw8CN1nVQHBFfloD+L3egixIa4FvUP6RosSA=
go.uber.org/atomic v1.4.0 h1:cxzIVoETapQEqDhQu3QfnvXAV4AlzcvUCxkVUFw3+EU=
go.uber.org/multierr v1.1.0 h1:HoEmRHQPVSqub6w2z2d2EOVs2fjyFRGyofhKuyDq0QI=
go.uber.org/zap v1.10.0 h1:ORx85nbTijNz8ljznvCMR1ZBIPKFn3jQrag10X2AsuM=
k8s.io/apiextensions-apiserver v0.17.2 h1:cP579D2hSZNuO/rZj9XFRzwJNYb41DbNANJb6Kolpss=
k8s.io/apimachinery v0.17.2 h1:hwDQQFbdRlpnnsR64Asdi55GyCaIP/3WQpMmbNBeWr4=
k8s.io/apiserver v0.17.2 h1:NssVvPALll6SSeNgo1Wk1h2myU1UHNwmhxV0Oxbcl8Y=
k8s.io/api v0.17.2 h1:NF1UFXcKN7/OOv1uxdRz3qfra8AHsPav5M93hlV9+Dc=
k8s.io/client-go v0.17.2 h1:ndIfkfXEGrNhLIgkr0+qhRguSD3u6DCmonepn1O6NYc=
k8s.io/cloud-provider v0.17.2 h1:5JVAaJ3JIlaXlOqAX/LkdnXPl1UlcjHMPbepo8LFcyo=
k8s.io/component-base v0.17.2 h1:0XHf+cerTvL9I5Xwn9v+0jmqzGAZI7zNydv4tL6Cw6A=
k8s.io/klog v1.0.0 h1:Pt+yjF5aB1xDSVbau4VsWe+dQNzA0qv1LlXdC2dF6Q8=
k8s.io/kube-controller-manager v0.17.2 h1:YqE2AM6YzSppRHjmxHSZn6FiXNy6hTpRPEa2SYpHV5s=
k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a h1:UcxjrRMyNx/i/y8G7kPvLyy7rfbeuf1PYyBf973pgyU=
k8s.io/kubernetes v1.17.2 h1:g1UFZqFQsYx88xMUks4PKC6tsNcekxe0v06fcVGRwVE=
k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 h1:p0Ai3qVtkbCG/Af26dBmU0E1W58NID3hSSh7cMyylpM=
sigs.k8s.io/controller-runtime v0.5.1 h1:TNidCfVoU/cs2i+9xoTcL/l7yhl0bDhYXU0NCG6wmiE=
sigs.k8s.io/sig-storage-lib-external-provisioner v4.1.0+incompatible h1:pp7GUmQZKI57EjGnjkY88V4QbVuMpkw/ijKXqL67EsI=
sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06 h1:zD2IemQ4LmOcAumeiyDWXKUI2SO0NYDe3H6QGvPOVgU=
sigs.k8s.io/structured-merge-diff v1.0.1-0.20191108220359-b1b620dd3f06 h1:zD2IemQ4LmOcAumeiyDWXKUI2SO0NYDe3H6QGvPOVgU=
sigs.k8s.io/yaml v1.1.0 h1:4A07+ZFc2wgJwo8YNlQpr1rVlgUDlxXHhPJciaPY5gs=
"""

# This is a set of workarounds that prevents the invocation of certain 'go list' commands by gazelle.
# When the go list commands are run for packages within these modules, they fail for reasons like:
# "things got renamed and now the master branch and the requested version have different package structures"
# Since gazelle only needs to use 'go list' to determine module boundaries, we can just specify particular
# module boundaries when we invoke go_repository, and avoid the whole problem.
KNOWN_IMPORTS = [
    # needed because of the unusual things kubernetes does to handle splitting out one monorepo into a bunch of separate repositories
    "k8s.io/kubernetes",
    # needed because of https://github.com/bazelbuild/bazel-gazelle/issues/545 affecting github.com/googleapis/gnostic/OpenAPIv2
    "github.com/googleapis/gnostic",
    # needed because of https://github.com/kubernetes/client-go/issues/749
    "k8s.io/client-go",
    # these two are needed because something's broken in etcd w/r/t renaming modules
    "go.etcd.io/etcd",
    "github.com/coreos/go-systemd",
    # unclear why this is needed
    "golang.org/x/sys",
    # needed because of a repository rename
    "github.com/improbable-eng/thanos",
]

def rook_dependencies():
    args = [
        "-exclude=vendor",
    ]
    for importpath in KNOWN_IMPORTS:
        args += ["-known_import", importpath]

    for line in ROOK_DEPS.split("\n"):
        if not line.strip():
            continue
        importpath, version, sum = line.split(" ")
        go_repository(
            name = importpath_to_repo_name(importpath),
            version = version,
            sum = sum,
            importpath = importpath,
            build_external = "external",
            build_extra_args = args,
            build_file_proto_mode = "disable_global",
            prepatch_cmds = [
                "find -name BUILD -delete",
                "find -name BUILD.bazel -delete",
                "rm -rf vendor",
            ],
        )

    tini_dependencies()

    go_repository(
        name = "com_github_rook_rook",
        build_extra_args = args,
        version = "v1.3.3",
        sum = "h1:L28rGUcqexFOu72WrstS8K09PRumLBXVjArnWnAp8bU=",
        importpath = "github.com/rook/rook",
        patch_cmds = [
            "echo \"exports_files(['toolbox.sh'])\" >images/ceph/BUILD.bazel",
            "echo \"exports_files(['csi/template', 'monitoring'])\" >cluster/examples/kubernetes/ceph/BUILD.bazel",
        ],
    )
