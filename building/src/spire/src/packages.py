# this file is derived from code previously in build-iso written by dmssargent

import os
import subprocess
import tempfile
import urllib.parse
import urllib.request
import urllib.error

import command
import resource
import util
from version import get_apt_branch

APT_REPO_BASE = "http://"


def verify_gpg_signature(data: bytes, signature: bytes, keyring: bytes) -> bool:
    with tempfile.TemporaryDirectory() as f:
        signaturefile = os.path.join(f, "data.gpg")
        datafile = os.path.join(f, "data")
        keyringfile = os.path.join(f, "keyring")

        util.writefile(signaturefile, signature)
        util.writefile(datafile, data)
        util.writefile(keyringfile, keyring)

        verify_command = ["gpg", "--no-default-keyring", "--keyring", keyringfile, "--verify", signaturefile, datafile]
        retcode = subprocess.call(verify_command, stderr=subprocess.DEVNULL, stdout=subprocess.DEVNULL)

        return retcode == 0


def fetch_url(url: str) -> bytes:
    with urllib.request.urlopen(url) as response:
        return response.read()


def fetch_url_and_check_hash(url: str, sha256hash: str) -> bytes:
    data = fetch_url(url)
    found = util.sha256sum_data(data)
    if found != sha256hash:
        command.fail("wrong hash: expected %s but got %s from url %s" % (sha256hash, found, url))
    return data


def fetch_signed_url(url: str, signature_url: str, keyring_resource: str) -> bytes:
    signature = fetch_url(signature_url)
    data = fetch_url(url)

    keyring = resource.get_resource(keyring_resource)
    if not verify_gpg_signature(data, signature, keyring):
        command.fail("signature verification FAILED on %s!" % url)

    return data


def parse_apt_kvs(data: str) -> dict:
    kvs = {}
    lines = data.split("\n")
    while lines:
        line = lines.pop()
        if not line.strip(): continue
        value_lines = []
        while line[0] == " ":
            value_lines.append(line[1:])
            if not lines:
                command.fail("malformed apt key/value file (reached start of file with indent)")
            line = lines.pop()
        if line.count(":") == 1 and line[-1] == ":":
            key = line[:-1]
        else:
            if ": " not in line:
                command.fail("malformed apt key/value line: could not extract key from '%s'" % line)
            key, first_value = line.split(": ", 1)
            value_lines.append(first_value)
        value_lines.reverse()  # we collected these backward, but want to output them in the right order
        if key in kvs:
            command.fail("duplicate key: %s=(%s,%s)" % (key, kvs[key], "\n".join(value_lines)))
        kvs[key] = "\n".join(value_lines)
    return kvs


def parse_apt_hash_list(section):
    hashes_by_path = {}

    for line in section.split("\n"):
        if line.count(" ") != 2:
            command.fail("found incorrectly formatted sha256 section")
        hashed, _, path = line.split(" ")
        hashes_by_path[path] = hashed

    return hashes_by_path


def download_and_verify_package_list(baseurl: str, dist: str="homeworld",
                                     keyring_resource: str="homeworld-archive-keyring.gpg") -> (str, dict):
    baseurl = baseurl.rstrip("/")
    url = baseurl + "/dists/" + dist

    release = fetch_signed_url(url + "/Release", url + "/Release.gpg", keyring_resource)
    packages_relpath = "main/binary-amd64/Packages"

    kvs = parse_apt_kvs(release.decode())
    if "SHA256" not in kvs:
        command.fail("cannot find section for sha256 hashes")

    hashes_by_path = parse_apt_hash_list(kvs["SHA256"])

    if packages_relpath not in hashes_by_path:
        command.fail("could not find hash for %s" % packages_relpath)

    packages = fetch_url_and_check_hash(url + "/" + packages_relpath, hashes_by_path[packages_relpath])

    parsed_packages = parse_apt_kv_list(packages.decode(), "Package")

    return baseurl, parsed_packages


def parse_apt_kv_list(data: str, sort_key: str) -> dict:
    lookup = {}
    for section in data.split("\n\n"):
        if not section.strip():
            continue
        kvs = parse_apt_kvs(section)
        if sort_key not in kvs:
            command.fail("expected to find key '%s' in section, but none was found" % sort_key)
        key = kvs[sort_key]
        if key in lookup:
            command.fail("duplicate entry %s='%s'" % (sort_key, key))
        lookup[key] = kvs
    return lookup


def download_package(package_name: str, verified_package_info: (str, dict)) -> (str, bytes):
    baseurl, verified_package_dict = verified_package_info
    if package_name not in verified_package_dict:
        command.fail("could not find package %s" % package_name)
    package_kvs = verified_package_dict[package_name]
    if "SHA256" not in package_kvs:
        command.fail("could not find sha256 hash for package %s" % package_name)
    sha256_hash = package_kvs["SHA256"]
    if "Filename" not in package_kvs:
        command.fail("could not find filename for package %s" % package_name)
    filename = package_kvs["Filename"]

    return os.path.basename(filename), fetch_url_and_check_hash(baseurl + "/" + filename, sha256_hash)


def verified_download_full(package_list: tuple) -> dict:
    """Download all the packages from the specified list from the apt branch, including verifying them.

    Returns a mapping of {package_name: (short_filename, package_bytes), ...}"""
    apt_branch = get_apt_branch()
    apt_url = APT_REPO_BASE + apt_branch
    try:
        verified_info = download_and_verify_package_list(apt_url)
        return {package_name: download_package(package_name, verified_info) for package_name in package_list}
    except urllib.error.HTTPError:
        command.fail("unable to access apt branch",
            "do you have an apt branch at %s?" % apt_url)
