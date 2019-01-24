import json
import os
import shutil
import subprocess
import tempfile

import aptbranch


def gs_rsync(local_path: str, remote_path: str, boto_path: str):
    # TODO: do this directly rather than by shelling out
    env = dict(os.environ)
    env["BOTO_PATH"] = boto_path
    subprocess.check_call(["gsutil", "-h", "Cache-Control:private, max-age=0, no-transform", "-m", "rsync", "-d", "-r", "-c", local_path, remote_path], env=env)


BOTO_PATH = "/homeworld/boto-key"

BOTO_TEMPLATE = """
[Credentials]
gs_service_key_file = %s
[Boto]
https_validate_certificates = True
[GSUtil]
content_language = en
default_api_version = 2
default_project_id = %s
""".lstrip()


def upload_gs(staging, root: str, branch_config: aptbranch.Config):
    upload_path = branch_config.upload_config['gcs-target']
    if not os.path.exists(BOTO_PATH):
        raise Exception("you need to put the GCP service account private key file into {}".format(BOTO_PATH))
    botoconfig_name = os.path.join(staging, "boto.config")
    with open(botoconfig_name, "w") as bout:
        with open(BOTO_PATH, "r") as f:
            project_id = json.load(f)["project_id"]
        bout.write(BOTO_TEMPLATE % (BOTO_PATH, project_id))
        bout.flush()
    gs_rsync(root, upload_path, botoconfig_name)


def upload_rsync(staging, root: str, branch_config: aptbranch.Config):
    target = branch_config.upload_config['rsync-target']
    subprocess.check_call(["rsync", "-avzc", "--progress", "--delete-delay", "--", root + "/", target])


UPLOAD_FUNCS = {
    "google-cloud-storage": upload_gs,
    "rsync": upload_rsync
}


def perform_uploads(uploads: dict, branch_config: aptbranch.Config) -> None:
    with tempfile.TemporaryDirectory() as staging:
        root = os.path.join(staging, "root")
        for remote_path, local_path in uploads.items():
            target = os.path.join(root, remote_path.lstrip("/"))
            if not os.path.isdir(os.path.dirname(target)):
                os.makedirs(os.path.dirname(target))
            shutil.copy2(local_path, target)

        upload_method = branch_config.upload_config["method"]
        if upload_method not in UPLOAD_FUNCS:
            raise Exception("unrecognized upload method %s" % upload_method)
        UPLOAD_FUNCS[upload_method](staging, root, branch_config)
