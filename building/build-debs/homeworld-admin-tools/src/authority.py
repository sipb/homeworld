import os

import command
import configuration
import resource
import util
import subprocess
import tempfile
import tarfile


def get_targz_path(check_exists=True):
    authorities = os.path.join(configuration.get_project(), "authorities.tgz")
    if check_exists and not os.path.exists(authorities):
        command.fail("authorities.tgz does not exist (run spire authority gen?)")
    return authorities


def generate() -> None:
    authorities = get_targz_path(check_exists=False)
    if os.path.exists(authorities):
        command.fail("authorities.tgz already exists")
    # tempfile.TemporaryDirectory() creates the directory with 0o600, which protects the private keys
    with tempfile.TemporaryDirectory() as d:
        certdir = os.path.join(d, "certdir")
        keyserver_yaml = os.path.join(d, "keyserver.yaml")
        util.writefile(keyserver_yaml, configuration.get_keyserver_yaml().encode())
        os.mkdir(certdir)
        try:
            subprocess.check_call(["keygen", keyserver_yaml, certdir, "supervisor-nodes"])
        except FileNotFoundError as e:
            if e.filename == "keygen":
                command.fail("could not find keygen binary. is the homeworld-keyserver dependency installed?")
            else:
                raise e
        subprocess.check_call(["tar", "-C", certdir, "-czf", authorities, "."])
        subprocess.check_call(["shred", "--"] + os.listdir(certdir), cwd=certdir)


def get_key_by_filename(keyname) -> bytes:
    authorities = get_targz_path()
    with tarfile.open(authorities, mode="r:gz") as tar:
        with tar.extractfile(keyname) as f:
            out = f.read()
            assert type(out) == bytes
            return out


main_command = command.mux_map("commands about cluster authorities", {
    "gen": command.wrap("generate authorities keys and certs", generate),
})
