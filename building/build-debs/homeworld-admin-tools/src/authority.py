import os

import command
import config
import resource
import util
import subprocess
import tempfile
import tarfile


def generate() -> None:
    authorities = os.path.join(config.get_project(), "authorities.tgz")
    if os.path.exists(authorities):
        command.fail("authorities.tgz already exists")
    with tempfile.TemporaryDirectory() as d:
        certdir = os.path.join(d, "certdir")
        keyserver_yaml = os.path.join(d, "keyserver.yaml")
        util.writefile(keyserver_yaml, config.get_keyserver_yaml().encode())
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


def get_authority_key(keyname) -> bytes:
    authorities = os.path.join(config.get_project(), "authorities.tgz")
    if not os.path.exists(authorities):
        command.fail("authorities.tgz does not exist (run spire authority gen?)")
    with tarfile.open(authorities, mode="r:gz") as tar:
        with tar.extractfile(keyname) as f:
            out = f.read()
            assert type(out) == bytes
            return out


main_command = command.mux_map("commands about cluster authorities", {
    "gen": command.wrap("generate authorities keys and certs", generate),
})
