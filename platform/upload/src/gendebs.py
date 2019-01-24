import sys
import os
import json
import subprocess
import tempfile
import hashlib
import tarfile

output, taroutput, keyid, *debs = sys.argv[1:]
assert debs

uploads = {}

distributions = """
Origin: Homeworld
Label: Homeworld
Suite: stretch
Codename: homeworld
Version: 9.0
Architectures: amd64
Components: main
Description: Homeworld code-deployment repository
SignWith: {keyid}
Update: homeworld
""".lstrip().format(keyid=keyid)


def sha256(filename):
    with open(filename, "rb") as f:
        h = hashlib.sha256()
        while True:
            data = f.read(65536)
            if not data:
                break
            h.update(data)
        return h.hexdigest()


def strip_name(x):
    assert x.startswith("bazel-out/k8-fastbuild/")
    return x.split("/",2)[2]


hash_to_deb = {}
for deb in debs:
    debhash = sha256(deb)
    assert debhash not in hash_to_deb
    hash_to_deb[debhash] = deb

# TODO: avoid copying everything
with tempfile.TemporaryDirectory() as staging:
    os.mkdir(staging + "/conf")
    with open(staging + "/conf/distributions", "w") as f:
        f.write(distributions)
    subprocess.check_call(["reprepro", "--verbose", "--basedir", staging, "includedeb", "homeworld"] + debs)

    # generate 'upload' entries from pool data
    for root, dirs, files in os.walk(os.path.join(staging, "pool")):
        for filename in files:
            source = os.path.join(root, filename)
            rel = os.path.relpath(source, staging)
            uploads[rel] = "file:" + strip_name(hash_to_deb[sha256(source)])

    # generate 'dists' entries
    for root, dirs, files in os.walk(os.path.join(staging, "dists")):
        for filename in files:
            source = os.path.join(root, filename)
            rel = os.path.relpath(source, staging)
            uploads[rel] = "tar:" + rel

    with tarfile.open(taroutput, "w") as tf:
        tf.add(os.path.join(staging, "dists"), arcname="dists")

with open(output, "w") as f:
    json.dump(uploads, f)
