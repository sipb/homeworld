import authority
import command
import datetime

import configuration
import resource
import os
import tempfile
import subprocess
import util
import packages
import keycrypt
from version import get_git_version

PACKAGES = ("homeworld-apt-setup", "homeworld-knc", "homeworld-keysystem", "homeworld-prometheus-node-exporter", \
            "homeworld-hyperkube", "homeworld-rkt", "homeworld-etcd", "homeworld-services", "homeworld-bootstrap-registry", \
            "homeworld-prometheus", "homeworld-etcd-metrics-exporter", "homeworld-auth-monitor")

# TODO: refactor this file to be more maintainable

# converts debian-9.2.0-amd64-mini.iso into a "cdpack", which can be more easily remastered
def regen_cdpack(source_iso, dest_cdpack):
    with tempfile.TemporaryDirectory() as d:
        loopdir = os.path.join(d, "loopdir")
        os.mkdir(loopdir)
        cddir = os.path.join(d, "cd")
        os.mkdir(cddir)
        subprocess.check_call(["bsdtar", "-xf", source_iso, "-C", cddir])
        subprocess.check_call(["chmod", "+w", "--recursive", cddir])
        subprocess.check_call(["gunzip", os.path.join(cddir, "initrd.gz")])
        subprocess.check_call(["tar", "-czf", dest_cdpack, "-C", d, os.path.basename(cddir)])


def add_password_to_log(password, creation_time):
    passwords = os.path.join(configuration.get_project(), "passwords")
    if not os.path.isdir(passwords):
        os.mkdir(passwords)
    passfile = os.path.join(passwords, "at-%s.gpg" % creation_time)
    util.writefile(passfile, keycrypt.gpg_encrypt_in_memory(password))


def list_passphrases():
    passwords = os.path.join(configuration.get_project(), "passwords")
    if not os.path.isdir(passwords):
        command.fail("no passwords stored")
    print("Passphrases:")
    for passfile in os.listdir(passwords):
        if passfile.startswith("at-") and passfile.endswith(".gpg"):
            date = passfile[3:-4]
            passph = keycrypt.gpg_decrypt_to_memory(os.path.join(passwords, passfile)).decode()
            print("   ", date, "=>", passph)
    print("End of list.")


def gen_iso(iso_image, authorized_key, cdpack=None):
    with tempfile.TemporaryDirectory() as d:
        inclusion = []

        util.copy(authorized_key, os.path.join(d, "authorized.pub"))
        util.writefile(os.path.join(d, "keyservertls.pem"), authority.get_pubkey_by_filename("./server.pem"))
        resource.copy_to("postinstall.sh", os.path.join(d, "postinstall.sh"))
        os.chmod(os.path.join(d, "postinstall.sh"), 0o755)
        inclusion += ["authorized.pub", "keyservertls.pem", "postinstall.sh"]

        for variant in configuration.KEYCLIENT_VARIANTS:
            util.writefile(os.path.join(d, "keyclient-%s.yaml" % variant), configuration.get_keyclient_yaml(variant).encode())
            inclusion.append("keyclient-%s.yaml" % variant)

        resource.copy_to("sshd_config", os.path.join(d, "sshd_config.new"))

        preseeded = resource.get_resource("preseed.cfg.in")
        generated_password = util.pwgen(20)
        creation_time = datetime.datetime.now().isoformat()
        git_hash = get_git_version().encode()
        add_password_to_log(generated_password, creation_time)
        print("generated password added to log")
        preseeded = preseeded.replace(b"{{HASH}}", util.mkpasswd(generated_password))
        preseeded = preseeded.replace(b"{{BUILDDATE}}", creation_time.encode())
        preseeded = preseeded.replace(b"{{GITHASH}}", git_hash);
        util.writefile(os.path.join(d, "preseed.cfg"), preseeded)

        inclusion += ["sshd_config.new", "preseed.cfg"]

        for package_name, (short_filename, package_bytes) in packages.verified_download_full(PACKAGES).items():
            assert "/" not in short_filename, "invalid package name: %s for %s" % (short_filename, package_name)
            assert short_filename.startswith(package_name + "_"), "invalid package name: %s for %s" % (short_filename, package_name)
            assert short_filename.endswith("_amd64.deb"), "invalid package name: %s for %s" % (short_filename, package_name)
            util.writefile(os.path.join(d, short_filename), package_bytes)
            inclusion.append(short_filename)

        if cdpack is not None:
            subprocess.check_call(["tar", "-C", d, "-xzf", cdpack, "cd"])
        else:
            subprocess.check_output(["tar", "-C", d, "-xz", "cd"], input=resource.get_resource("debian-9.2.0-cdpack.tgz"))

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
    "passphrases": command.wrap("decrypt a list of passphrases used by recently-generated ISOs", list_passphrases),
})
