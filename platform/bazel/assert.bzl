load("//bazel:util.bzl", "escape")

def build_assert(name, src, message, condition, deps = None, visibility = None):
    cmdbase = "if ( {} ); then " + \
              "    cp $(location {}) $@; " + \
              "else " + \
              "    echo ASSERT FAILED: {} 1>&2; " + \
              "    false; " + \
              "fi"
    cmdline = cmdbase.format(condition, src, escape(message))
    native.genrule(
        name = name + "-assert",
        srcs = [src] + (deps or []),
        outs = [name],
        cmd = cmdline,
        visibility = visibility,
    )
