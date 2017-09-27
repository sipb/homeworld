import os

import command
import config
import resource
import util
import subprocess
import tempfile


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


main_command = command.mux_map("commands about cluster authorities", {
    "gen": command.wrap("generate authorities keys and certs", generate),
})
