load("@bazel_tools//tools/build_defs/repo:git.bzl", "new_git_repository")

def tini_dependencies():
    new_git_repository(
        name = "tini",
        commit = "fec3683b971d9c3ef73f284f176672c44b448662",
        remote = "https://github.com/krallin/tini",
        shallow_since = "1524295900 +0200",
        build_file_content = """filegroup(name = "source", srcs = glob(["**"]), visibility = ["//visibility:public"])""",
    )
