load("@bazel_tools//tools/build_defs/pkg:pkg.bzl", "pkg_tar", "pkg_deb")
load("//bazel:version.bzl", "hash_compute", "version_compute")
load("//bazel:oci_digest.bzl", "oci_digest")
load("@io_bazel_rules_docker//container:container.bzl", "container_image")

def homeworld_deb(name, package, bin=None, data=None, deps=None, depends=None, prerm=None, postrm=None, preinst=None, postinst=None, visibility=None):
    pkg_tar(
        name = name + "-contents-bin",
        files = bin or {},
        mode = "0755",
    )
    pkg_tar(
        name = name + "-contents-data",
        files = data or {},
        mode = "0644",
    )
    if not deps:
        deps = []
    pkg_tar(
        name = name + "-contents",
        deps = [
            name + "-contents-bin",
            name + "-contents-data",
        ] + deps,
    )

    # these three are all visible so that generate_version_cache can get this info
    hash_compute(
        name = name + "-hash",
        inputs = [
            name + "-contents",
            prerm,
            postrm,
            preinst,
            postinst,
        ],
        strings = [
            package,
            " ".join(depends or []),
        ],
        visibility = visibility,
    )
    native.genrule(
        name = name + "-name-rule",
        outs = [name + "-name"],
        cmd = "echo " + _escape(package) + " >$@",
        visibility = visibility,
    )
    version_compute(
        name = name + "-version",
        package = package,
        hashfile = name + "-hash",
        visibility = visibility,
    )

    pkg_deb(
        name = name,
        data = name + "-contents",
        package = package,
        architecture = "amd64",
        maintainer = "Hyades Maintenance <sipb-hyades@mit.edu>",
        version_file = name + "-version",
        description = "Code deployment package",
        section = "misc",
        depends = depends,
        visibility = visibility,
        prerm = prerm,
        postrm = postrm,
        preinst = preinst,
        postinst = postinst,
    )

def _escape(s):
    return "'" + s.replace("'", "'\"'\"'") + "'"

def aci_manifest(name, aciname, version_file, ports=None, exec=None, visibility=None):
    cmdline = "./$(location //bazel:aci-manifest-gen) $(location " + version_file + ") " + _escape(aciname)
    if exec != None:
        if type(exec) != type([]):
            fail("exec parameter to aci_manifest must be a list, or None")
        for x in exec:
            cmdline += " " + _escape(x)
    cmdline += " --"
    if ports:
        for portname, portinfo in ports.items():
            if ":" not in portinfo:
                portinfo = "tcp:" + portinfo
            cmdline += " " + _escape(portname) + ":" + _escape(portinfo)
    cmdline += " >\"$@\""
    native.genrule(
        name = name,
        srcs = [version_file],
        tools = ["//bazel:aci-manifest-gen"],
        outs = [name + ".json"],
        cmd = cmdline,
        visibility = visibility,
    )

def homeworld_aci(name, aciname, bin=None, data=None, deps=None, aci_dep=None, ports=None, exec=None, visibility=None):
    pkg_tar(
        name = name + "-contents-bin",
        files = bin or {},
        mode = "0755",
    )
    pkg_tar(
        name = name + "-contents-data",
        files = data or {},
        mode = "0644",
    )
    tar_deps = [
        name + "-contents-bin",
        name + "-contents-data",
    ]
    if deps:
        tar_deps += deps
    if aci_dep:
        tar_deps += [aci_dep + "-rootfs"]

    # for recursive inclusion
    pkg_tar(
        name = name + "-rootfs",
        deps = tar_deps,
        visibility = visibility,
    )

    # for actual use (has the /rootfs prefix)
    pkg_tar(
        name = name + "-rootdir",
        deps = tar_deps,
        package_dir = "/rootfs",
    )

    # these three are all visible so that generate_version_cache can get this info
    hash_compute(
        name = name + "-hash",
        inputs = [
            name + "-rootdir",
        ],
        strings = [
            aciname,
        ] + [
            "x" + exec_i for exec_i in (exec or [])
        ] + [
            "p" + portname for portname in (ports or {}).keys()
        ] + [
            "P" + portinfo for portinfo in (ports or {}).values()
        ],
        visibility = visibility,
    )
    native.genrule(
        name = name + "-name-rule",
        outs = [name + "-name"],
        cmd = "echo " + _escape(aciname) + " >$@",
        visibility = visibility,
    )
    version_compute(
        name = name + "-version",
        package = aciname,
        hashfile = name + "-hash",
        visibility = visibility,
    )

    aci_manifest(
        name = name + "-manifest",
        aciname = aciname,
        version_file = name + "-version",
        exec = exec,
        ports = ports,
    )
    pkg_tar(
        name = name,
        extension = "tar.gz",
        files = {
            name + "-manifest" : "manifest",
        },
        deps = [name + "-rootdir"],
        mode = "0644",
        visibility = visibility,
    )

def homeworld_oci(name, bin=None, data=None, deps=None, oci_dep=None, ports=None, exec=None, visibility=None):
    pkg_tar(
        name = name + "-contents-bin",
        files = bin or {},
        mode = "0755",
    )
    pkg_tar(
        name = name + "-contents-data",
        files = data or {},
        mode = "0644",
    )
    tar_deps = [
        name + "-contents-bin",
        name + "-contents-data",
    ]
    if deps:
        tar_deps += deps

    container_image(
        name = name,
        repository = "homeworld.private",
        base = oci_dep,
        cmd = exec,
        tars = tar_deps,
        ports = ports,
        visibility = visibility,
    )

    # version references are provided by <name>.ocidigest, which is the sha256:AAA....AAA identifier that we can use to
    # pin a particular image for deployment.
    oci_digest(
        name = name + ".ocidigest",
        image = name,
        visibility = visibility,
    )

    # to facilitate our rather unfortunate tag workaround, we need a hash-only digest, not including the
    # sha256: prefix. this is a stop-gap solution and should be removed once we've switched away from rkt.
    native.genrule(
        name = name + ".shortdigest-rule",
        srcs = [name + ".ocidigest"],
        outs = [name + ".shortdigest"],
        cmd = "cut -d ':' -f 2 <'$<' >'$@'",
        visibility = visibility,
    )

def pythonize(name, zip, visibility=None):
    native.genrule(
        name = name + "-rule",
        srcs = [zip],
        outs = [name],
        cmd = "echo '#!/usr/bin/env python3' | cat - $< >$@",
        visibility = visibility,
    )
