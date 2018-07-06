import datetime
import os
import re
import tempfile

import actions
import context
import validate
import aptbranch

# NOTE: /h/ is the chroot link to the homeworld/building/ directory of the git repo.

# The root of the directory tree that built debian packages are placed in.
BIN_DIR_BASE = "/h/binaries/"
# The expected name of a project's glassfile.
GLASSFILE = "glass.yaml"
# Path used for filesystem-based tempdirs, when /tmp isn't large enough.
ALTERNATE_TEMPDIR_LOCATION = "/fstemp/"
# Location of glassfile schema.
GLASS_SCHEMA_NAME = "project-schema.yaml"


def get_bindir(apt_branch):
    """Get the directory in which the built packages for a certain apt branch are placed."""
    return os.path.join(BIN_DIR_BASE, apt_branch)


def aci_name(name: str, full_version: str) -> str:
    return "%s-%s-linux-amd64.aci" % (name, full_version)


def get_output_path_for_aci(branch: str, name: str, full_version: str):
    return os.path.join(get_bindir(branch), aci_name(name, full_version))


def log(operation, *message):
    print(("[%s]" % operation).rjust(13), " ".join(map(str, message)))


class Project:
    def __init__(self, path):
        self.path = os.path.abspath(path)
        glasspath = os.path.join(self.path, GLASSFILE)
        if not os.path.exists(glasspath):
            raise Exception("cannot find glassfile under project: %s" % path)
        self.glassfile = validate.load_validated(glasspath, GLASS_SCHEMA_NAME)
        self.pkgbase = os.path.basename(self.path)

    @property
    def change_date(self):
        return datetime.datetime.strptime(self.glassfile["control"]["date"], "%Y-%m-%dT%H:%M:%S%z")

    @property
    def full_version(self):
        return self.glassfile["control"]["version"]

    @property
    def base_version(self):
        return self.full_version.split("-")[0]

    @property
    def package_name(self):
        if self.build_type == "deb":
            return "%s_%s_amd64.deb" % (self.pkgbase, self.full_version)
        elif self.build_type == "aci":
            return aci_name(self.pkgbase, self.full_version)
        elif self.build_type == "tgz":
            return "%s-%s.tgz" % (self.pkgbase, self.full_version)
        else:
            raise Exception("cannot get package name for unknown build type: %s" % self.build_type)

    def get_output_path(self, branch: str) -> str:
        return os.path.join(get_bindir(branch), self.package_name)

    def is_built(self, branch: str) -> bool:
        """Returns whether the built package exists for a branch."""
        return os.path.exists(self.get_output_path(branch))

    def ensure_create_bin_path(self, branch):
        """Ensures that the directory for package output (for this branch) is created, or create it if it doesn't exist."""
        base_bin = os.path.dirname(self.get_output_path(branch))

        if not os.path.isdir(base_bin):
            os.makedirs(base_bin)

    def ensure_package_name_matches(self):
        package_name = self.glassfile["control"]["name"]
        if package_name != self.pkgbase:
            raise Exception("mismatch between directory (%s) and declared name (%s)" % (self.pkgbase, package_name))

    @property
    def build_type(self) -> str:
        return self.glassfile["control"]["type"]

    def clean(self, branch_config: aptbranch.Config) -> bool:
        branch = branch_config.name
        if self.build_type == "folder":
            projects = self.scan_subprojects()
            any = False
            for project in projects:
                any |= project.clean(branch_config)
            return any
        elif self.is_built(branch):
            log("glass", "cleaning", self.package_name, "on", branch)
            path = self.get_output_path(branch)
            if os.path.exists(path + ".asc"):
                os.unlink(path + ".asc")
            os.unlink(path)
            return True
        else:
            return False

    def scan_subprojects(self) -> list:
        packages = {name for name in os.listdir(self.path) if os.path.isdir(os.path.join(self.path, name))}
        ordering = []
        for stage in self.glassfile["stages"]:
            pattern = re.compile(stage["pattern"])
            found = sorted(package for package in packages if pattern.match(package))
            if stage.get("build", True):
                ordering += found
            for package in found:
                packages.remove(package)
        return [Project(os.path.join(self.path, package)) for package in ordering]

    def run(self, branch_config: aptbranch.Config, debug=False) -> None:
        branch = branch_config.name
        if self.build_type == "folder":
            # not a direct build; rather a directory thereof
            projects = self.scan_subprojects()
            log("glass", "found", len(projects), "packages")
            for project in projects:
                project.run(branch_config)
            log("glass", "finished building", len(projects), "packages")
            return

        if self.is_built(branch):
            log("glass", "package", self.package_name, "already built for", branch)
            return

        log("glass", "*********************")
        log("glass", "beginning build of package", self.package_name, "for", branch)
        if debug:
            log("debug", "running with debug mode on")

        self.ensure_create_bin_path(branch)
        self.ensure_package_name_matches()

        os.unsetenv("GOROOT")

        tempdir = None  # use default

        if not self.glassfile["control"].get("use-tmpfs", True):
            # if use-tmpfs is false specifically, this means that the staging directory needs to be placed on a normal
            # filesystem, probably because the build process needs too much space
            tempdir = ALTERNATE_TEMPDIR_LOCATION
            if not os.path.isdir(tempdir):
                os.makedirs(tempdir)
            log("glass", "using alternate tempdir location", tempdir)

        assert debug in (True, False, None), "invalid selection for debug parameter"

        with tempfile.TemporaryDirectory(prefix="glass-", dir=tempdir) as d:
            ctx = context.Context(self, os.path.join(d, "stage"), os.path.join(d, "tree"), tempdir, branch_config)
            try:
                if not os.path.isdir(ctx.stagedir):
                    os.makedirs(ctx.stagedir)
                if not os.path.isdir(ctx.outputdir):
                    os.makedirs(ctx.outputdir)

                for step in self.glassfile["build"]:
                    actions.invoke_action(ctx, step)
                actions.invoke_final(ctx, self.build_type, self.glassfile["control"])
            except Exception as e:
                if debug:
                    log("debug", "dropping to debug shell due to error:", e)
                    actions.perform_debug_shell(ctx)
                raise e

        if self.is_built(branch):
            log("glass", "package", self.package_name, "built for", branch)
        else:
            raise Exception("internal error: package failed to build: %s" % self.package_name)
