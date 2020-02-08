def _escape(s):
    return "'" + s.replace("$", "$$").replace("'", "'\"'\"'") + "'"

def _generate(name, tool, arguments, inputs, visibility = None):
    native.genrule(
        name = name + "-rule",
        srcs = inputs,
        tools = [tool],
        outs = [name],
        cmd = "./$(location " + tool + ") " + " ".join(arguments) + " >\"$@\"",
        visibility = visibility,
    )

def version_compute(name, package, hashfile, visibility = None):
    cmdline = [
        "$(location //:VERSION)",
        "$(location //upload:version-cache)",
        _escape(package),
        "$(location " + hashfile + ")",
    ]
    _generate(
        name = name,
        tool = "//bazel:version-compute",
        arguments = cmdline,
        inputs = ["//:VERSION", "//upload:version-cache", hashfile],
        visibility = visibility,
    )

def hash_compute(name, inputs, strings, visibility = None):
    cmdline = [
        ("$(location " + input + ")" if input else "--empty")
        for input in inputs
    ] + ["--"] + strings
    _generate(
        name = name,
        tool = "//bazel:hash-compute",
        arguments = cmdline,
        inputs = [input for input in inputs if input],
        visibility = visibility,
    )

def generate_version_cache(name, debs, visibility = None):
    targets = []
    for artifact in debs:
        targets += [
            artifact + "-name",
            artifact + "-hash",
            artifact + "-version",
        ]
    _generate(
        name = name,
        tool = "//bazel:cache-compute",
        arguments = ["$(location " + entry + ")" for entry in targets],
        inputs = targets,
        visibility = visibility,
    )
