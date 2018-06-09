import os
import subprocess
import tempfile

# The directory in which build helpers are kept.
HELPER_DIR = "/h/binaries"


def get_go_package(branch, version):
    return os.path.join(HELPER_DIR, branch, "helper-go-%s.tgz" % version)


def unpack(branch, version, destination):
    os.makedirs(destination)
    subprocess.check_call(["tar", "-C", destination, "--strip-components=1", "-xf", get_go_package(branch, version), "go"])


def build(branch, version, sources, output, updateenv=None, ldflags=None):
    with tempfile.TemporaryDirectory(prefix="gobuild-") as gobin:
        unpack(branch, version, os.path.join(gobin, "go"))

        env = os.environ
        if updateenv is not None:
            env.update(updateenv)
        env["GOROOT"] = os.path.join(gobin, "go")
        env["PATH"] = os.path.join(gobin, "go/bin") + ":" + env["PATH"]

        checkver = subprocess.check_output(["go", "version"], stderr=subprocess.DEVNULL).strip().decode()
        if checkver != "go version go%s linux/amd64" % version:
            raise Exception("golang version mismatch: %s" % checkver)

        flags = ["-o", output]
        if ldflags is not None:
            flags += ["-ldflags", ldflags]
        subprocess.check_call(["go", "build"] + flags + ["--"] + sources)
