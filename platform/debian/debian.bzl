# TODO: include the snapshot'd security repository as well
# TODO: hardening by checking release date

load("//debian:debclean.bzl", "debclean")

def _escape(s):
    return "'" + s.replace("'", "'\"'\"'") + "'"

def debootstrap_fetch(name, release, mirror, variant = None, extra = None, visibility = None):
    """Fetch packages and store them as a tarball."""

    debootstrap_cmd = ["/usr/sbin/debootstrap", "--make-tarball=\"$@\"", "--components=main"]
    if variant:
        debootstrap_cmd.append(_escape("--variant=" + variant))
    if extra:
        debootstrap_cmd.append(_escape("--include=" + ",".join(extra)))
    debootstrap_cmd += [_escape(release), "$${BTEMP}", _escape(mirror)]

    cmds = [
        "BTEMP=\"$$(mktemp -d)\"",
        " ".join(debootstrap_cmd),
        "rm -rf \"$${BTEMP}\"",
    ]
    native.genrule(
        name = name + "-rule",
        outs = [name],
        cmd = "\n".join(cmds),
        visibility = visibility,
    )

def debootstrap_unpack(name, tarball, release, mirror, variant = None, extra = None, visibility = None):
    """Unpack packages from tarball (or mirror as fallback) into a new installation; provide new install as tarball."""

    debootstrap_cmd = ["fakeroot", "/usr/sbin/debootstrap", "--unpack-tarball=\"$$(realpath \"$<\")\"", "--components=main"]
    if variant:
        debootstrap_cmd.append(_escape("--variant=" + variant))
    if extra:
        debootstrap_cmd.append(_escape("--include=" + ",".join(extra)))
    debootstrap_cmd += ["--foreign", _escape(release), "$${BTEMP}", _escape(mirror)]

    cmds = [
        "BTEMP=\"$$(mktemp -d)\"",
        " ".join(debootstrap_cmd),
        "tar -czf '$@' --hard-dereference -C \"$${BTEMP}\" $$(ls -A \"$${BTEMP}\")",
        "rm -rf \"$${BTEMP}\"",
    ]
    native.genrule(
        # ~28 seconds
        name = name + "-rule",
        outs = [name],
        srcs = [tarball],
        cmd = "\n".join(cmds),
        local = 1,
        visibility = visibility,
    )

CLEAN_FAKECHROOT = "//debian:clean_fakechroot.py"

def _debremove(root, packages, force_depends = None):
    # extra path needed for the commands that dpkg runs
    cmd = ["PATH=$$PATH:/usr/local/sbin:/usr/sbin:/sbin", "fakeroot", "fakechroot", "dpkg", "--purge", "--root=" + root]
    cmd += ["--force-remove-essential", "--no-triggers"]
    if force_depends:
        cmd.append("--force-depends")
    cmd += [_escape(package) for package in packages]
    return " ".join(cmd)

def debootstrap_configure(name, partial, remove = None, remove_dpkg = None, visibility = None):
    """Configure debian installation kept within specified tarball, possibly removing some packages."""

    # NOTE: if you get fakechroot errors, that might mean you're bootstrapping too old of a version of debian
    cmds = [
        "BTEMP=\"$$(mktemp -d)\"",
        "tar -xzf '$<' -C \"$${BTEMP}\"",
        "DEBOOTSTRAP_DIR=\"$${BTEMP}/debootstrap\" fakeroot fakechroot /usr/sbin/debootstrap --second-stage --second-stage-target \"$${BTEMP}\"",
    ]
    if remove:
        cmds.append(
            _debremove("\"$${BTEMP}\"", remove),
        )
    if remove_dpkg:
        cmds += [
            _debremove("\"$${BTEMP}\"", ["perl-base", "debconf"], force_depends = True),
            _debremove("\"$${BTEMP}\"", ["dpkg"], force_depends = True),
        ]
    cmds += [
        "$(location " + CLEAN_FAKECHROOT + ") \"$${BTEMP}\"",
        "tar -czf '$@' --hard-dereference -C \"$${BTEMP}\" $$(ls -A \"$${BTEMP}\")",
        "rm -rf \"$${BTEMP}\"",
    ]
    native.genrule(
        name = name + "-rule",
        outs = [name],
        srcs = [partial],
        tools = [CLEAN_FAKECHROOT],
        cmd = "\n".join(cmds),
        local = 1,
        visibility = visibility,
    )

# mirror = "http://snapshot.debian.org/archive/debian/20180710T043017Z/"
# variant = "minbase"
# release = "stretch"
def debootstrap(name, release, mirror, variant = None, extra = None, remove = None, remove_dpkg = None, clean_opts = None, visibility = None):
    """Download, unpack, configure, and clean a debian installation."""

    debootstrap_fetch(
        name = name + "-sources.tgz",
        release = release,
        mirror = mirror,
        extra = extra,
        variant = variant,
    )
    debootstrap_unpack(
        name = name + "-foreign.tgz",
        tarball = name + "-sources.tgz",
        release = release,
        mirror = mirror,
        variant = variant,
        extra = extra,
    )
    if not clean_opts:
        config_name = name
    else:
        config_name = name + "-unclean.tgz"
        debclean(
            name = name,
            partial = config_name,
            clean_opts = clean_opts,
            visibility = visibility,
        )
    debootstrap_configure(
        name = config_name,
        partial = name + "-foreign.tgz",
        visibility = visibility,
        remove = remove,
        remove_dpkg = remove_dpkg,
    )

def debinstall(name, base, packages, visibility = None):
    if not packages:
        native.alias(name = name, actual = base)
    else:
        cmds = [
            "BTEMP=\"$$(mktemp -d)\"",
            "tar -xzf '$<' -C \"$${BTEMP}\"",
            "fakeroot fakechroot chroot \"$${BTEMP}\" apt-get update",
            "fakeroot fakechroot chroot \"$${BTEMP}\" apt-get install -y -- " + " ".join([_escape(pkg) for pkg in packages]),
            "$(location " + CLEAN_FAKECHROOT + ") \"$${BTEMP}\"",
            "tar -czf '$@' --hard-dereference -C \"$${BTEMP}\" $$(ls -A \"$${BTEMP}\")",
            "rm -rf \"$${BTEMP}\"",
        ]
        native.genrule(
            name = name + "-preclean.tgz-rule",
            outs = [name + "-preclean.tgz"],
            srcs = [base],
            tools = [CLEAN_FAKECHROOT],
            cmd = "\n".join(cmds),
            local = 1,
        )
        debclean(
            name = name,
            partial = name + "-preclean.tgz",
            clean_opts = ["apt_files"],
            visibility = visibility,
        )
