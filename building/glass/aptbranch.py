"This module handles resolving which apt branches exist, and what their corresponding signing keys are."

import os
import re
import subprocess

import validate

CONFIG_PATH = "/h/apt-branch-config/branches.yaml"
CONFIG_SCHEMA_NAME = "branches-schema.yaml"


def get_env_branch():
    """Get the apt branch named in the environment variable ${HOMEWORLD_APT_BRANCH}. If not found, return None."""

    branch = os.getenv("HOMEWORLD_APT_BRANCH")

    if not branch:
        return None

    if not re.match("^[0-9a-zA-Z_.-]+/[0-9a-zA-Z_.-]+$", branch):
        print("apt branch should be of the form <username>/<branch>")
        print("allowed characters: [0-9a-zA-Z_.-]")
        raise Exception("invalid apt branch: %s" % branch)

    return branch


def check_signing_key(key_id):
    "Throw an exception if the specified key key doesn't exist in the gpg keyring."

    if subprocess.call(["gpg", "--list-keys", key_id], stdout=subprocess.DEVNULL) != 0:
        if branch == "root/master":
            print("If you're basing this build off the master branch, import its signing key with")
            print('gpg --import building/apt-branch-config/default-key.asc')
        raise Exception("apt signing key not in gpg keyring: %s" % branch)


def export_key(keyid, armor=False):
    "Exports the specified key from gpg and returns the export."
    result = subprocess.check_output(["gpg", "--export"] + (["--armor"] if armor else []) + ["--", keyid])
    if not result.strip():
        raise Exception("empty result from gpg for keyid: '%s'" % keyid)
    return result


def select_branch_config(branches_config: list, branch: str):
    for config in branches_config:
        if config["name"] == branch:
            return config
    raise Exception("no config found for %s in %s" % (branch, CONFIG_PATH))


class Config:
    def __init__(self, branch: str):
        if not os.path.exists(CONFIG_PATH):
            raise Exception("cannot find branches config at %s, use %s.example to create one" % (CONFIG_PATH, CONFIG_PATH))
        branches_config = validate.load_validated(CONFIG_PATH, CONFIG_SCHEMA_NAME)["branches"]
        self.config = select_branch_config(branches_config, branch)

        check_signing_key(self.signing_key)

    @property
    def name(self) -> str:
        return self.config["name"]

    @property
    def signing_key(self) -> str:
        return self.config["signing-key"]

    @property
    def apt_url(self):
        url_prefix = self.config.get("apt-url-prefix", "")
        if url_prefix and not url_prefix.endswith("/"):
            url_prefix += "/"
        return url_prefix + self.name
