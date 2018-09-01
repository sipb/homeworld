import os
import shutil
import subprocess

import acbuild
import actions  # yes, importing itself!
import aptbranch
import debclean
import debuild
import gobuild
import project
import util
from context import Context


def invoke_action(ctx: Context, step_object):
    step_type = step_object["type"]
    step_object = {k.replace("-", "_"): v for k, v in step_object.items() if k != "type"}
    # finds the right method based on the step name
    # if this doesn't always succeed, something's wrong with the schema
    getattr(actions, "perform_%s" % step_type.replace("-", "_"))(ctx, **step_object)


def invoke_final(ctx: Context, build_type, control):
    getattr(actions, "final_%s" % build_type)(ctx, control)


def perform_copy(context: Context, input: str=None, stage: str=None, output: str=None, recursive: bool = False,
                 follow_symlinks: bool = True) -> None:
    input = context.input(input, allow_none=True)
    stage = context.stage(stage, create_parents=True, allow_none=True)
    output = context.output(output, create_parents=True, allow_none=True)
    paths = [path for path in [input, stage, output] if path is not None]
    if len(paths) != 2:
        raise Exception("copy step expects two of {input, stage, output}!")
    source, dest = paths
    project.log("copy", context.namepath(source), "to", context.namepath(dest))

    if recursive:
        if os.path.isdir(dest):
            if os.listdir(dest):
                raise Exception("cannot recursively copy into directory with existing contents")
            for rel in os.listdir(source):
                shutil.copytree(os.path.join(source, rel), os.path.join(dest, rel), copy_function=util.copy3,
                                symlinks=follow_symlinks)
        else:
            if os.path.exists(dest):
                raise Exception("cannot recursively copy over existing file")
            shutil.copytree(source, dest, copy_function=util.copy3, symlinks=follow_symlinks)
    else:
        util.copy3(source, dest)


def perform_upstream_extract(context: Context, upstream: str, version: str, stage: str, focus: str = None) -> None:
    upstream = upstream.replace("%", version)
    project.log("upstream", "unpacking archive", upstream)
    tarfile = context.upstream(upstream)
    targetdir = context.stage(stage.replace("%", version))
    if not os.path.isdir(targetdir):
        os.makedirs(targetdir)
    if focus is not None:
        focus = os.path.normpath(focus.replace("%", version))
        strip_count = focus.strip("/").count("/") + 1
        subprocess.check_call(["tar", "-C", targetdir, "-xf", tarfile, "--strip-components=%d" % strip_count, focus])
    else:
        subprocess.check_call(["tar", "-C", targetdir, "-xf", tarfile])


def perform_upstream(context: Context, upstream: str, version: str, stage: str):
    upstream = upstream.replace("%", version)
    project.log("upstream", "importing", upstream)
    input = context.upstream(upstream)
    stage = context.stage(stage.replace("%", version), create_parents=True)
    shutil.copyfile(input, stage)


def perform_remove(context: Context, stage: str, recursive: bool = False):
    project.log("remove", "removing directory" if recursive else "removing file", stage)
    path = context.stage(stage)
    if recursive:
        shutil.rmtree(path)
    else:
        os.remove(path)


def prepare_exec_source_target(context: Context, kind: str, input: str, stage: str, output: str):
    source = context.input(input, allow_none=True)
    dest = context.output(output, create_parents=True, allow_none=True)

    if stage is not None:
        if dest is None:
            dest = context.stage(stage, create_parents=True, allow_none=True)
        elif source is None:
            source = context.stage(stage, require_existence=True, allow_none=True)
        else:
            raise Exception(kind + " step expects at most two of {input, stage, output}")

    if source is not None and dest is not None:
        project.log(kind, "generating", context.namepath(dest), "from", context.namepath(source))
    elif source is not None:
        project.log(kind, "processing", context.namepath(source))
    elif dest is not None:
        project.log(kind, "generating", context.namepath(dest))
    else:
        project.log(kind, "running snippet")

    if source is not None:
        with open(source, "r") as f:
            source = f.read()

    return source, dest


def perform_python(context: Context, code: str, input: str = None, stage: str = None, output: str = None) -> None:
    sourcedata, target = prepare_exec_source_target(context, "python", input, stage, output)
    env = {
        "aptbranch": aptbranch,
        "context": context,
        "project": context.project,
    }
    if sourcedata is not None:
        env["input"] = sourcedata
    if target is not None:
        # this selectively allows 'return' only when a result is expected.
        code = "def _expr():\t" + code.replace("\n", "\n\t")
    # Yeah, this is eval. Remember: don't run untrusted build scripts! There's no sandboxing.
    locals = {}
    exec(code, env, locals)
    if target is not None:
        result = locals["_expr"]()
        if type(result) != bytes:
            result = str(result).encode()
        with open(target, "wb") as out:
            out.write(result)


def perform_bash(context: Context, code: str, input: str = None, stage: str = None, output: str = None) -> None:
    sourcedata, target = prepare_exec_source_target(context, "bash", input, stage, output)

    env = dict(os.environ)
    env.update({
        "INPUT": context.inputdir,
        "STAGE": context.stagedir,
        "OUTPUT": context.outputdir,
        "FULL_VERSION": context.project.full_version,
        "BASE_VERSION": context.project.base_version
    })

    with context.tempfile() as f:
        f.write(b"set -e -u\n")
        f.write(code.encode())
        f.flush()
        stdout = subprocess.PIPE if target is not None else None
        proc = subprocess.run(["bash", f.name], cwd=context.stagedir, env=env, input=sourcedata, stdout=stdout,
                              check=True)
        if target is not None:
            with open(target, "wb") as out:
                out.write(proc.stdout)


def perform_go_build(context: Context, version: str, stage: str, sources_input: list = (), packages: list = (),
                     gopath: str = "go", no_cgo: bool = False, ldflags: str = None):
    if not sources_input and not packages:
        raise Exception("go-build expects at least one source file or package to build")
    project.log("go", "(%s)" % version, "compiling output", stage)
    env = {"GOPATH": context.stage(gopath, require_existence=True), "CGO_ENABLED": "0" if no_cgo else "1"}
    output = context.stage(stage, create_parents=True)
    sources = [context.input(source) for source in sources_input] + list(packages)
    gobuild.build(context.branch, version, sources, output, env, ldflags)


def perform_go_prepare(context: Context, version: str, stage: str):
    project.log("go", "(%s)" % version, "preparing for external build")
    gobuild.unpack(context.branch, version, context.stage(stage))


def perform_debootstrap(context: Context, release: str, version: str, stage: str, extra: list = ()):
    project.log("debootstrap", "bootstrapping debian release", release, "at version", version, "with", len(extra),
                "extra packages")
    # NOTE: if this fails with a fakechroot error, that probably means that you should update the version of debian that
    # you're bootstrapping.
    args = ["fakeroot", "fakechroot", "debootstrap", "--components=main", "--variant=minbase"]
    if extra:
        args += ["--include=" + ",".join(extra)]
    args += [release, context.stage(stage, create_parents=True),
             "http://snapshot.debian.org/archive/debian/%s/" % version]
    subprocess.check_call(args)
    # TODO: hardening by checking release date


def perform_debremove(context: Context, packages: list, stage: str, force_remove_essential: bool = False,
                      force_depends: bool = False, no_triggers: bool = False):
    if not packages:
        raise Exception("expected at least one package to be specified for debremove")
    project.log("debremove", "removing", len(packages), "debian packages")
    args = ["fakeroot", "fakechroot", "dpkg", "--purge"]
    args += ["--root=" + context.stage(stage, require_existence=True)]
    if force_remove_essential:
        args += ["--force-remove-essential"]
    if force_depends:
        args += ["--force-depends"]
    if no_triggers:
        args += ["--no-triggers"]
    args += packages
    subprocess.check_call(args)


def perform_debinstall(context: Context, packages: list, stage: str):
    # TODO: move as many checks as possible into schema validation
    if not packages:
        raise Exception("expected at least one package to be specified for debinstall")
    project.log("debinstall", "installing", len(packages), "debian packages")

    rootfs = context.stage(stage, require_existence=True)
    chroot_base = ["fakeroot", "fakechroot", "chroot", rootfs]

    subprocess.check_call(chroot_base + ["apt-get", "update"])
    subprocess.check_call(chroot_base + ["apt-get", "install", "-y", "--"] + packages)
    debclean.clean_apt_files(rootfs)


def perform_debclean(context: Context, stage: str, options: list) -> None:
    if not options:
        raise Exception("expected at least one option to be specified for debclean")
    for option in options:
        if option not in debclean.DEBCLEAN_OPTIONS:
            raise Exception("invalid debclean option: %s" % option)
    project.log("debclean", "performing cleaning with options:", *options)
    rootfs = context.stage(stage, require_existence=True)
    for opt in options:
        debclean.DEBCLEAN_OPTIONS[opt](rootfs)


def perform_fakechroot_clean(context: Context, stage: str) -> None:
    """cleans up any symbolic links pointing with absolute paths to the build directory itself"""
    project.log("fakechroot-clean", "cleaning up directory:", stage)
    rootfs = context.stage(stage, require_existence=True)
    for root, dirs, files in os.walk(rootfs):
        for f in files:
            path = os.path.join(root, f)
            if not os.path.islink(path):
                continue
            full_link = os.readlink(path)
            if not os.path.isabs(full_link):
                continue
            rootrel = os.path.relpath(full_link, rootfs)
            if rootrel.split("/")[0] == "..":
                # doesn't point within the rootfs; nothing to do
                continue
            os.remove(path)
            os.symlink(os.path.join("/", rootrel), path)


def perform_mkdir(context: Context, stage: str = None, output: str = None, recursive: bool = False) -> None:
    if stage is not None:
        assert output is None
        project.log("mkdir", stage)
        target = context.stage(stage)
    else:
        assert output is not None, "schema should ensure this"
        target = context.output(output)
    project.log("mkdir", context.namepath(output))
    if recursive:
        os.makedirs(target)
    else:
        os.mkdir(target)


def perform_debug_shell(context: Context) -> None:
    project.log("debug", "launching shell")
    subprocess.check_call(["bash"], cwd=context.stagedir)
    project.log("debug", "closed shell")


def perform_aci_unpack(context: Context, name: str, version: str, stage: str = None, output: str = None):
    if [stage, output].count(None) != 1:
        raise Exception("aci-unpack expects exactly one of {stage, output} to be specified")
    if stage is None:
        targetdir = context.output(output, create_parents=True)
    else:
        targetdir = context.stage(stage, create_parents=True)
    project.log("aci", "unpacking container rootfs from", name, "version", version, "to", context.namepath(targetdir))
    if not os.path.isdir(targetdir):
        os.makedirs(targetdir)
    aci = project.get_output_path_for_aci(context.branch, name, version)
    subprocess.check_call(["tar", "-C", targetdir, "-xf", aci, "--strip-components=1", "rootfs"])


def perform_acbuild(context: Context, name: str, stage: str, exec: str=None, copy: list=(), env: dict=None, mounts: dict=None, labels: dict=None, ports: list=()):
    project.log("acbuild", "building container", name)
    with acbuild.Build(context.branch) as build:
        build.set_name(name)
        if exec is not None:
            build.set_exec(exec)
        for copyentry in copy:
            output, input, stageent = copyentry["output"], copyentry.get("input"), copyentry.get("stage")
            if [input, stageent].count(None) != 1:
                raise Exception("acbuild/copy must have either input or stage, not both!")
            if input is not None:
                build.copy_in(context.input(input), output)
            else:
                build.copy_in(context.stage(stageent), output)
        if env is not None:
            for key, value in env.items():
                build.env_add(key, value)
        if mounts is not None:
            for key, value in mounts.items():
                build.mount_add(key, value)
        if labels is not None:
            for key, value in labels.items():
                build.label_add(key, value)
        if ports is not None:
            for port in ports:
                build.port_add(port["name"], port["protocol"], port["port"])
        build.write(context.stage(stage))


def final_deb(context: Context, control: dict) -> None:
    project.log("debuild", context.project.pkgbase, "version", context.project.full_version)
    depends = control.get("depends", [])
    install_scripts = control.get("install-scripts", {})
    install_scripts = {key: context.input(value) for key, value in install_scripts.items()}

    debuild.perform_debuild(context.outputdir, context.project.pkgbase, context.project.full_version,
                            context.project.change_date, depends, context.project.get_output_path(context.branch),
                            install_scripts)


def final_aci(context: Context, control: dict) -> None:
    project.log("acbuild", context.project.pkgbase, "version", context.project.full_version)
    with acbuild.Build(context.branch) as b:
        b.set_name("homeworld.private/" + context.project.pkgbase)
        if "set-exec" in control:
            b.set_exec(*control["set-exec"].split(" "))
        b.label_add("version", context.project.full_version)
        for x in os.listdir(context.outputdir):
            b.copy_in(os.path.join(context.outputdir, x), "/" + x)
        for port in control.get("ports", []):
            b.port_add(port["name"], port["protocol"], port["port"])
        b.write(context.project.get_output_path(context.branch))


def final_tgz(context: Context, control: dict):
    project.log("tar", "creating", context.project.pkgbase, "version", context.project.full_version)
    subprocess.check_call(["tar", "-C", context.outputdir, "-czf", context.project.get_output_path(context.branch), "--"] + os.listdir(context.outputdir))
