"This module handles resolving which apt branches exist, and what their corresponding signing keys are."

import os
import re
import subprocess

SETUP_DIR = "/h/setup-apt-branch/"


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


def load_key_value_pairs(filename):
    "Load a mapping of key/value pairs (with key and value separated by whitespace) from a file."

    with open(filename, "r") as f:
        kvs = {}
        for line in f:
            if not line.strip():
                continue
            parts = line.strip().split()
            if len(parts) != 2:
                raise Exception("invalid line in file %s: '%s'" % (filename, line))
            key, value = parts
            kvs[key] = value
        return kvs


def get_signing_key_unchecked(branch):
    "Get the signing key ID for a specific apt branch, or throw an exception if not found. Does not verify that the key actually exists in the gpg keyring."

    for filename in ["signing-keys", "global-signing-keys"]:
        path = os.path.join(SETUP_DIR, filename)
        if filename == "signing-keys" and not os.path.exists(path):
            continue
        kvs = load_key_value_pairs(path)
        if branch in kvs:
            signing_key = kvs[branch]
            if not re.match("^[0-9a-zA-Z]+$", signing_key):
                raise Exception("apt signing key invalid: %s" % signing_key)
            return signing_key
    raise Exception("apt branch %s not found in signing keys" % branch)


def get_signing_key(branch):
    "Get the signing key ID for a specific apt branch. Throw an exception if not found, or if the key doesn't exist in the gpg keyring."

    key_id = get_signing_key_unchecked(branch)
    if subprocess.call(["gpg", "--list-keys", key_id], stdout=subprocess.DEVNULL) != 0:
        if branch == "root/master":
            print("If you're basing this build off the master branch, import its signing key with")
            print('gpg --import "$(readlink -f "${APT_SETUP_DIR}/../upload-debs/default-repo-signing-key.gpg")"')
        raise Exception("apt signing key not in gpg keyring: %s" % branch)
    return key_id


def export_key(keyid, armor=False):
    "Exports the specified key from gpg and returns the export."
    result = subprocess.check_output(["gpg", "--export"] + (["--armor"] if armor else []) + ["--", keyid])
    if not result.strip():
        raise Exception("empty result from gpg for keyid: '%s'" % keyid)
    return result
