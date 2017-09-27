import authority
import command
import time

import config
import resource
import os
import tempfile
import subprocess
import util

PACKAGES = ("homeworld-apt-setup", "homeworld-knc", "homeworld-keysystem")

# TODO: refactor this file to be more maintainable


# converts debian-9.0.0-amd64-mini.iso into a "cdpack", which can be more easily remastered
def regen_cdpack(source_iso, dest_cdpack):
    with tempfile.TemporaryDirectory() as d:
        loopdir = os.path.join(d, "loopdir")
        os.mkdir(loopdir)
        cddir = os.path.join(d, "cd")
        os.mkdir(cddir)
        subprocess.check_call(["sudo", "mount", "-o", "loop", "--", source_iso, loopdir])
        try:
            subprocess.check_call(
                ["rsync", "--quiet", "--archive", "--hard-links", "--exclude=TRANS.TBL", loopdir + "/", cddir])
        finally:
            subprocess.check_call(["sudo", "umount", loopdir])
        subprocess.check_call(["chmod", "+w", "--recursive", cddir])
        subprocess.check_call(["gunzip", os.path.join(cddir, "initrd.gz")])
        subprocess.check_call(["tar", "-czf", dest_cdpack, "-C", d, os.path.basename(cddir)])


def parse_version_from_debian_changelog(debian_folder, package_name):
    changelog = util.readfile(os.path.join(debian_folder, "changelog")).decode()
    current_version_line = changelog.split("\n")[0]
    if "(" not in current_version_line:
        command.fail("invalid changelog file for %s" % package_name)
    version = current_version_line.split("(")[1].split(")")[0]
    if "/" in version:
        command.fail("invalid version for %s" % package_name)
    return version


def resolve_packages(repo_building_folder: str, package_names=PACKAGES):
    build_debs = os.path.join(repo_building_folder, "build-debs")
    if not os.path.isdir(build_debs):
        command.fail("expected build-debs folder inside %s" % repo_building_folder)

    binary_package_paths = []
    for package_name in package_names:
        version = parse_version_from_debian_changelog(os.path.join(build_debs, package_name, "debian"), package_name)

        binary_package_path = os.path.join(build_debs, "binaries", "%s_%s_amd64.deb" % (package_name, version))
        if not os.path.exists(binary_package_path):
            command.fail("could not find binary package for %s at version %s" % (package_name, version))

        binary_package_paths.append(binary_package_path)

    return binary_package_paths


def gen_iso(iso_image, repo_building_folder, authorized_key, cdpack=None):
    binary_package_paths = resolve_packages(repo_building_folder)
    with tempfile.TemporaryDirectory() as d:
        inclusion = []

        util.copy(authorized_key, os.path.join(d, "authorized.pub"))
        util.writefile(os.path.join(d, "keyservertls.pem"), authority.get_authority_key("./server.pem"))
        resource.copy_to("postinstall.sh", os.path.join(d, "postinstall.sh"))
        inclusion += ["authorized.pub", "keyservertls.pem", "postinstall.sh"]

        for variant in config.KEYCLIENT_VARIANTS:
            util.writefile(os.path.join(d, "keyclient-%s.yaml" % variant), config.get_keyclient_yaml(variant).encode())
            inclusion.append("keyclient-%s.yaml" % variant)

        resource.copy_to("sshd_config", os.path.join(d, "sshd_config.new"))

        preseeded = resource.get_resource("preseed.cfg.in")
        generated_password = util.pwgen(20)
        print("generated password:", generated_password.decode())
        preseeded = preseeded.replace(b"{{HASH}}", util.mkpasswd(generated_password))
        util.writefile(os.path.join(d, "preseed.cfg"), preseeded)

        inclusion += ["sshd_config.new", "preseed.cfg"]

        for path in binary_package_paths:
            util.copy(path, os.path.join(d, os.path.basename(path)))

        inclusion += [os.path.basename(path) for path in binary_package_paths]

        if cdpack is not None:
            subprocess.check_call(["tar", "-C", d, "-xzf", cdpack, "cd"])
        else:
            subprocess.check_output(["tar", "-C", d, "-xz", "cd"], input=resource.get_resource("debian-9.0.0-cdpack.tgz"))

        subprocess.check_output(["cpio", "--create", "--append", "--format=newc", "--file=cd/initrd"],
                                input="".join("%s\n" % filename for filename in inclusion).encode(), cwd=d)
        subprocess.check_call(["gzip", os.path.join(d, "cd/initrd")])

        files_for_md5sum = subprocess.check_output(["find", ".", "-follow", "-type", "f", "-print0"], cwd=os.path.join(d, "cd")).decode().split("\0")
        assert files_for_md5sum.pop() == ""
        md5s = subprocess.check_output(["md5sum", "--"] + files_for_md5sum, cwd=os.path.join(d, "cd"))
        util.writefile(os.path.join(d, "cd", "md5sum.txt"), md5s)

        subprocess.check_call(["genisoimage", "-quiet", "-o", iso_image, "-r", "-J", "-no-emul-boot", "-boot-load-size", "4", "-boot-info-table", "-b", "isolinux.bin", "-c", "isolinux.cat", os.path.join(d, "cd")])


main_command = command.mux_map("commands about building installation ISOs", {
    "regen-cdpack": command.wrap("regenerate cdpack from upstream ISO", regen_cdpack),
    "gen": command.wrap("generate ISO", gen_iso),
})
