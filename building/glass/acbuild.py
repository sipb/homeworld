import os
import subprocess
import tempfile

import gobuild

ACBUILD_VER = "0.4.0"


def get_acbuild_package(branch, version):
    return os.path.join(gobuild.HELPER_DIR, branch, "helper-acbuild-%s.tgz" % version)


class Build:
    def __init__(self, branch, begin_from=None):
        self._tempdir_obj = tempfile.TemporaryDirectory(prefix="acbuild-")
        self._tempdir = self._tempdir_obj.name
        self._begin_from = begin_from
        acbuild_dir = os.path.join(self._tempdir, "acbuild")
        subprocess.check_call(["tar", "-C", self._tempdir, "-xf", get_acbuild_package(branch, ACBUILD_VER), "acbuild"])
        self._acbuild = os.path.join(acbuild_dir, "acbuild")

    def _exec(self, *params: str):
        subprocess.check_call([self._acbuild] + [str(param) for param in params], cwd=self._tempdir)

    def __enter__(self):
        if self._begin_from is not None:
            if not os.path.exists(self._begin_from):
                raise Exception("no such base ACI: %s" % self._begin_from)
            self._exec("begin", self._begin_from)
        else:
            self._exec("begin")
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self._exec("end")
        self._tempdir_obj.cleanup()

    def set_name(self, name):
        self._exec("set-name", "--", name)

    def set_exec(self, path: str, *elements: str):
        self._exec("set-exec", "--", path, *elements)

    def copy_in(self, source_path, dest_path):
        self._exec("copy", "--", os.path.abspath(source_path), dest_path)

    def env_add(self, env_key, env_value):
        self._exec("environment", "add", "--", env_key, env_value)

    def mount_add(self, mountname, mountpoint):
        txpath = os.path.join(self._tempdir, ".acbuild/currentaci/rootfs", mountpoint.lstrip("/"))
        subprocess.check_call(["mkdir", "-p", "--", txpath])
        self._exec("mount", "add", "--", mountname, mountpoint)

    def label_add(self, key, value):
        self._exec("label", "add", "--", key, value)

    def port_add(self, portname, proto, portnum):
        self._exec("port", "add", "--", portname, proto, portnum)

    def write(self, output_file):
        self._exec("write", output_file)
