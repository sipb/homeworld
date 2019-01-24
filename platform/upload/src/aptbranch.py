"This module handles resolving which apt branches exist, and what their corresponding signing keys are."

import os
import subprocess

import validate


CONFIG_SCHEMA_NAME = os.path.join(os.path.dirname(__file__), "branches-schema.yaml")


def check_signing_key(key_id):
    "Throw an exception if the specified key key doesn't exist in the gpg keyring."

    if subprocess.call(["gpg", "--list-keys", key_id], stdout=subprocess.DEVNULL) != 0:
        raise Exception("apt signing key not in gpg keyring: %s" % key_id)


def select_branch_config(path: str, branches_config: list, branch: str):
    for config in branches_config:
        if config["name"] == branch:
            return config
    raise Exception("no config found for %s in %s" % (branch, path))


class Config:
    @classmethod
    def load_configs(cls, path: str):
        return validate.load_validated(path, CONFIG_SCHEMA_NAME)

    @classmethod
    def list_branches(cls, path: str):
        return [config['name'] for config in cls.load_configs(path)['branches']]

    def __init__(self, path: str, branch: str):
        configs = Config.load_configs(path)
        branches_config = configs["branches"]
        self.upload_targets = configs.get("upload-targets", [])
        self.config = select_branch_config(path, branches_config, branch)

        check_signing_key(self.signing_key)

    @property
    def name(self) -> str:
        return self.config["name"]

    @property
    def signing_key(self) -> str:
        return self.config["signing-key"]

    @property
    def download(self) -> str:
        return self.config["download"]

    @property
    def upload_config(self):
        try:
            return self.config["upload"]
        except KeyError as e:
            raise Exception('no upload configuration for branch {}'.format(self.name)) from e
