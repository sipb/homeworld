load("@bazel_skylib//lib:paths.bzl", "paths")

def sign(name, data, visibility=None):
    native.genrule(
        name = name,
        outs = [name + ".asc"],
        srcs = [data, "//upload:KEYID"],
        cmd = """gpg --armor --batch --no-tty --detach-sign --local-user "0x$$(cat "$(location //upload:KEYID)")" --output '$@' -- '$(location {})' """.format(data),
        visibility = visibility,
    )

def upload(name, acis=None, debs=None, visibility=None):
    data = []
    args = []
    if acis:
        acis_with_sigs = []
        for i, aci in enumerate(acis):
            sig = "{}-sig-{}".format(name, i)
            sign(
                name = sig,
                data = aci,
            )
            acis_with_sigs += [aci, sig]
        items = " ".join(['"$(location {})"'.format(item) for item in acis_with_sigs])

        ref = name + "-acis"
        args += ["$(location " + ref + ")"]
        data += [ref]
        native.genrule(
            name = ref + "-rule",
            outs = [ref],
            srcs = acis_with_sigs,
            tools = ["//upload:src/genacis.py"],
            cmd = """python3 $(location //upload:src/genacis.py) "$@" {}""".format(items),
        )
        data += acis_with_sigs
    else:
        args += ["--"]
    if debs:
        debs = [deb + ".deb" for deb in debs]  # ensure that we only get the actual *PACKAGES*
        items = " ".join(['"$(location {})"'.format(item) for item in debs])

        ref = name + "-debs"
        datref = name + "-dists.tar"
        args += ["$(location " + ref + ")", "$(location " + datref + ")"]
        data += [ref, datref]
        native.genrule(
            name = ref + "-rule",
            outs = [ref, datref],
            srcs = debs + ["//upload:KEYID"],
            tools = ["//upload:src/gendebs.py"],
            cmd = """python3 $(location //upload:src/gendebs.py) "$(location {})" "$(location {})" "$$(cat "$(location //upload:KEYID)")" {}""".format(ref, datref, items),
        )
        data += debs
    else:
        args += ["--", "--"]
    data += ["//upload:src/doupload.py"]

    # this shouldn't be required, but there's apparently some sort of runfiles collection bug...?
    # this is the hint from https://github.com/bazelbuild/bazel/issues/1147#issuecomment-428698802 -- but I'm not sure
    # if that particular bug is at all related.
    native.sh_library(
        name = name + "-lib",
        data = data,
    )

    native.sh_binary(
        name = name,
        srcs = ["//upload:src/wrapper.sh"],
        args = ["$(location //upload:src/doupload.py)"] + args + ["$(location //upload:branches.yaml)", "$(location //upload:BRANCH_NAME)"],
        deps = [name + "-lib"],
        data = data + ["//upload:branches.yaml", "//upload:BRANCH_NAME"],
        visibility = ["//visibility:public"],
    )
