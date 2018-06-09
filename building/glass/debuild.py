import datetime
import os
import shutil
import subprocess
import tempfile


def write_file(directory: str, dest: str, contents: str):
    with open(os.path.join(directory, dest), "w") as f:
        f.write(contents)
        if contents[-1:] != "\n":
            f.write("\n")


INSTALL_SCRIPT_NAMES = ("prerm", "postrm", "preinst", "postinst")


def perform_debuild(tree: str, name: str, version: str, date: datetime.datetime, dependencies: list, output: str,
                    install_scripts: dict):
    with tempfile.TemporaryDirectory(prefix="debuild-") as wrapdir:
        builddir = os.path.join(wrapdir, name)
        os.mkdir(builddir)

        debdir = os.path.join(builddir, "debian")
        os.mkdir(debdir)

        datestr = date.strftime("%a, %d %b %Y %H:%M:%S %z")
        write_file(debdir, "changelog", """
{name} ({version}) stretch; urgency=medium

  * Updated release

 -- Hyades Maintenance <sipb-hyades@mit.edu>  {date}""".strip().format(name=name, version=version, date=datestr))

        write_file(debdir, "compat", "10")

        write_file(debdir, "control", """
Source: {name}
Maintainer: Hyades Maintenance <sipb-hyades@mit.edu>
Section: misc
Priority: optional
Build-Depends: debhelper (>= 9)
Standards-Version: 4.1.1

Package: {name}
Architecture: any
Depends: ${{misc:Depends}}, ${{shlibs:Depends}}{dependencies}
Description: Code deployment package for {name}.
 This package is used for code deployment to Homeworld clusters.
""".strip().format(name=name, dependencies="".join(", %s" % dep for dep in dependencies)))

        write_file(debdir, "copyright", "")

        write_file(debdir, "rules", """
#!/usr/bin/make -f
%:
\tdh $@
override_dh_auto_install:
\tcp -R -t $$(pwd)/debian/{name}/ {source}/*
""".strip().format(name=name, source=os.path.abspath(tree)))
        os.chmod(os.path.join(debdir, "rules"), 0o755)

        source = os.path.join(debdir, "source")
        os.mkdir(source)
        write_file(source, "format", "1.0")

        write_file(debdir, "%s.dirs" % name,
                   "\n".join(os.path.relpath(tree, dirpath) for dirpath, _, _ in os.walk(tree)))

        for iname, ipath in install_scripts.items():
            if iname not in INSTALL_SCRIPT_NAMES:
                raise Exception("no such install script name: %s" % iname)
            with open(ipath, "r") as f:
                write_file(debdir, iname, f.read())

        # note: this should probably use the expanded --unsigned-source, --unsigned-changes, --build=binary options,
        # but using those seems to cause the command to fail inexplicably, so they aren't being used right now.
        subprocess.check_call(["debuild", "-us", "-uc", "-b"], cwd=builddir, stdout=subprocess.DEVNULL)

        shutil.copyfile(os.path.join(wrapdir, "%s_%s_amd64.deb" % (name, version)), output)
