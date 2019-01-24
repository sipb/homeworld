import json
import os
import sys
import tarfile
import tempfile
import upload
import aptbranch


def load_json(filename, default=None):
    if not filename or filename == "--":
        return default
    else:
        with open(filename) as f:
            return json.load(f)


def do_upload(acis, debs, debtar, branches_yaml, branch_name):
    uploads = load_json(acis, {})
    uploads.update(load_json(debs, {}))

    with tempfile.TemporaryDirectory() as staging:
        if debtar:
            with tarfile.open(debtar) as tf:
                # this is not secure; luckily, we trust the source of this archive!
                tf.extractall(staging)
        resolved = {}
        for k, v in uploads.items():
            ref, path = v.split(":")
            if ref == "tar":
                path = os.path.join(staging, path)
            elif ref == "file":
                path = path.split("/",1)[1]
            else:
                raise Exception("unrecognized: %s" % ref)
            assert os.path.exists(path), "no such file: %s" % path
            resolved[k] = path

        upload.perform_uploads(resolved, aptbranch.Config(branches_yaml, branch_name))


if __name__ == "__main__":
    acis, debs, debtar, branches_yaml, branch_name = sys.argv[1:]
    with open(branch_name, "r") as f:
        branch_name = f.read().strip()
    do_upload(acis, debs, debtar, branches_yaml, branch_name)
