load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@rules_foreign_cc//:workspace_definitions.bzl", "rules_foreign_cc_dependencies")

VERSION = "14.2.9"
SHA256 = "349e099292f6e2bbfc3b25d2b114b30a814d47694261ea8a72e1a13d840a707e"

def ceph_dependencies():
    rules_foreign_cc_dependencies()
    http_archive(
        name = "ceph",
        url = "https://download.ceph.com/tarballs/ceph_" + VERSION + ".orig.tar.gz",
        sha256 = SHA256,
        patch_cmds = [
            # remove symlinks that create cycles (which break glob)
            "find -name '.qa' -type l -delete",
            # remove filenames with ":" in them (which are disallowed in filegroups)
            "rm src/test/common/test_blkdev_sys_block/sys/dev/block/8:0 src/test/common/test_blkdev_sys_block/sys/dev/block/9:0",
            # remove filenames with special characters (which break bazel for some reason)
            "rm src/boost/libs/wave/test/testwave/testfiles/utf8-test-*/file.hpp",
            "rmdir src/boost/libs/wave/test/testwave/testfiles/utf8-test-*",
            # error out if any other filenames with ":" in them exist
            "find -name '*:*' | ( ! grep -q . )",
        ],
        strip_prefix = "ceph-" + VERSION + "/",
        build_file_content = """filegroup(name = "source", srcs = glob(["**"]), visibility = ["//visibility:public"])""",
    )
