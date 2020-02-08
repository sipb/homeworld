load("//bazel:util.bzl", "escape_inner")

def substitute(name, template, kvs = None, kfs = None, visibility = None):
    cmdline = "./$(location //bazel:substitute) $(location " + template + ")"
    if kvs:
        for k, v in kvs.items():
            if "=" in k or "<" in k:
                fail("cannot have = or < signs in keys")
            cmdline += " '" + escape_inner(k) + "=" + escape_inner(v) + "'"
    srcs = [template]
    if kfs:
        for k, f in kfs.items():
            if "=" in k or "<" in k:
                fail("cannot have = or < signs in keys")
            cmdline += " '" + escape_inner(k) + "<" + "$(location " + f + ")'"
            srcs += [f]
    cmdline += " >\"$@\""
    native.genrule(
        name = name + "-rule",
        srcs = srcs,
        tools = ["//bazel:substitute"],
        outs = [name],
        cmd = cmdline,
        visibility = visibility,
    )
