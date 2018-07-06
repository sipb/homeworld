import json
import os
import re
import shutil
import subprocess
import tempfile

import aptbranch
import project

# given a repository descriptor of:
#   HOMEWORLD_APT_BRANCH=hyades-deploy.celskeggs.com/test01
# the following would be populated into hyades-deploy.celskeggs.com
# (or, for rsync, <host>:<dir>/hyades-deploy.celskeggs.com/)
#
# /test01/
# /test01/dists/
# /test01/dists/[...]
# /test01/pool/
# /test01/pool/[...]
# /test01/aci/
# /test01/aci/homeworld.private/
# /test01/aci/homeworld.private/flannel-0.8.0-4-linux-amd64.aci
# /test01/aci/homeworld.private/flannel-0.8.0-4-linux-amd64.aci.asc


def upload(bindir: str, branch_config: aptbranch.Config) -> None:
    branch = branch_config.name
    keyid = branch_config.signing_key

    files = os.listdir(bindir)

    project.log("upload", "preparing uploads...")

    with tempfile.TemporaryDirectory() as tempdir:
        uploads = {}
        debs = []
        if any(file.endswith(".aci") for file in files):
            project.log("upload", "preparing upload of acis...")
        for file in files:
            path = os.path.join(bindir, file)
            if not os.path.isfile(path):
                raise Exception("not a normal file: %s" % file)
            if file.endswith(".aci"):
                upload_aci(path, uploads, keyid)
            elif file.endswith(".deb"):
                debs.append(path)
        project.log("upload", "preparing upload of debs...")
        upload_apt(debs, uploads, keyid, tempdir)

        project.log("upload", "performing", len(uploads), "uploads to", branch)
        perform_uploads(uploads, branch_config)
        project.log("upload", "upload to", branch, "complete!")


def upload_aci(path: str, uploads: dict, keyid: str):
    if not os.path.exists(path + ".asc"):
        subprocess.check_call(["gpg", "--armor", "--detach-sign", "--local-user", "0x" + keyid, path])
    uploads["/aci/homeworld.private/%s" % os.path.basename(path)] = path
    uploads["/aci/homeworld.private/%s.asc" % os.path.basename(path)] = path + ".asc"


distributions_base = """
Origin: Homeworld
Label: Homeworld
Suite: stretch
Codename: homeworld
Version: 9.0
Architectures: amd64
Components: main
Description: Homeworld code-deployment repository
SignWith: %s
Update: homeworld
""".lstrip()


def upload_apt(debs: list, uploads: dict, keyid: str, tempdir: str) -> None:
    basenames = {os.path.basename(deb).split("_")[0] for deb in debs}
    # the packages that get built differently on each apt branch
    if "homeworld-apt-setup" not in basenames:
        raise Exception("homeworld-apt-setup is not built!")
    if "homeworld-admin-tools" not in basenames:
        raise Exception("homeworld-admin-tools is not built!")
    staging = os.path.join(tempdir, "apt-stage")
    os.makedirs(os.path.join(staging, "conf"))
    with open(os.path.join(staging, "conf", "distributions"), "w") as f:
        f.write(distributions_base % keyid)
    subprocess.check_call(["reprepro", "--verbose", "--basedir", staging, "includedeb", "homeworld"] + debs)
    for subdir in ("dists", "pool"):
        for root, dirs, files in os.walk(os.path.join(staging, subdir)):
            for filename in files:
                source = os.path.join(root, filename)
                rel = os.path.relpath(source, staging)
                uploads[rel] = source


def gs_rsync(local_path: str, remote_path: str, boto_path: str):
    # TODO: do this directly rather than by shelling out
    env = dict(os.environ)
    env["BOTO_PATH"] = boto_path
    subprocess.check_call(["gsutil", "-h", "Cache-Control:private, max-age=0, no-transform", "-m", "rsync", "-d", "-r", "-c", local_path, "gs://" + remote_path], env=env)


GS_BRANCH_REGEX = re.compile("[a-z0-9-]+[.][a-z0-9.-]+/[a-z0-9-]+")


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
    branch = branch_config.name
    if not GS_BRANCH_REGEX.match(branch):
        raise Exception("not an uploadable branch: %s" % branch)
    if not os.path.exists(BOTO_PATH):
        raise Exception("you need to put the GCP service account private key file into /homeworld/boto-key")
    botoconfig_name = os.path.join(staging, "boto.config")
    with open(botoconfig_name, "w") as bout:
        with open(BOTO_PATH, "r") as f:
            project_id = json.load(f)["project_id"]
        bout.write(BOTO_TEMPLATE % (BOTO_PATH, project_id))
        bout.flush()
    gs_rsync(root, branch, botoconfig_name)


def upload_rsync(staging, root: str, branch_config: aptbranch.Config):
    host, path = branch_config.name.split('/', 1)
    target_dir = os.path.join(branch_config.upload_config["dir"], path)
    user = branch_config.upload_config["user"]
    if "@" in user or ":" in user:
        raise Exception("unsupported characters (@ or :) in upload user")
    dest = "%s@%s:%s" % (user, host, target_dir)
    subprocess.check_call(["rsync", "-avzc", "--progress", "--delete-delay", "--", root + "/", dest])


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
