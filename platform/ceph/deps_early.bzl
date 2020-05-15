load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

def ceph_dependencies_early():
    git_repository(
        name = "rules_foreign_cc",
        remote = "https://github.com/bazelbuild/rules_foreign_cc",
        commit = "ed3db61a55c13da311d875460938c42ee8bbc2a5",
        patches = [
            # we need this patch so that bazel's -D__DATE__="redacted" CFLAG doesn't become
            # -D__DATE__=redacted, which causes code that uses __DATE__ to break.
            "//ceph:foreign_cc/0001-fix-date-quoting.patch",
            # (see https://github.com/bazelbuild/rules_foreign_cc/issues/239
            # and https://github.com/bazelbuild/rules_foreign_cc/pull/362)

            # we need this so that we can correctly reference the generated libraries
            "//ceph:foreign_cc/0002-more-output-groups.patch",
            # see https://github.com/bazelbuild/rules_foreign_cc/issues/376
        ],
        shallow_since = "1574792034 +0100",
    )
