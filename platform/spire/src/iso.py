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
import setup
import gzip
from version import get_git_version

PACKAGES = ("homeworld-apt-setup",)

# TODO: refactor this file to be more maintainable


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


def mode_serial(includedir, cddir, inclusion):
    resource.copy_to("isolinux.cfg.serial", os.path.join(cddir, "isolinux.cfg"))


MODES={"serial": mode_serial}


def gen_iso(iso_image, authorized_key, mode=None):
    with tempfile.TemporaryDirectory() as d:
        inclusion = []

        with open(os.path.join(d, "dns_bootstrap_lines"), "w") as outfile:
            outfile.write(setup.dns_bootstrap_lines())

        inclusion += ["dns_bootstrap_lines"]
        util.copy(authorized_key, os.path.join(d, "authorized.pub"))
        util.writefile(os.path.join(d, "keyservertls.pem"), authority.get_pubkey_by_filename("./clusterca.pem"))
        resource.copy_to("postinstall.sh", os.path.join(d, "postinstall.sh"))
        os.chmod(os.path.join(d, "postinstall.sh"), 0o755)
        inclusion += ["authorized.pub", "keyservertls.pem", "postinstall.sh"]

        util.writefile(os.path.join(d, "keyserver.domain"), configuration.get_keyserver_domain().encode())
        inclusion.append("keyserver.domain")

        resource.copy_to("sshd_config", os.path.join(d, "sshd_config.new"))

        preseeded = resource.get_resource("preseed.cfg.in")
        generated_password = util.pwgen(20)
        creation_time = datetime.datetime.now().isoformat()
        git_hash = get_git_version().encode()
        add_password_to_log(generated_password, creation_time)
        print("generated password added to log")
        preseeded = preseeded.replace(b"{{HASH}}", util.mkpasswd(generated_password))
        preseeded = preseeded.replace(b"{{BUILDDATE}}", creation_time.encode())
        preseeded = preseeded.replace(b"{{GITHASH}}", git_hash)

        mirror = configuration.get_config().mirror
        if mirror.count("/") < 1 or mirror.count(".") < 1:
            command.fail("invalid mirror specification '%s'; must be of the form HOST.NAME/PATH")
        mirror_host, mirror_dir = mirror.split("/", 1)
        preseeded = preseeded.replace(b"{{MIRROR-HOST}}", mirror_host.encode())
        preseeded = preseeded.replace(b"{{MIRROR-DIR}}", ("/" + mirror_dir).encode())

        realm = configuration.get_config().realm
        preseeded = preseeded.replace(b"{{KERBEROS-REALM}}", realm.encode())

        cidr_nodes, upstream_dns_servers = configuration.get_config().cidr_nodes, configuration.get_config().dns_upstreams

        node_cidr_prefix = ".".join(str(cidr_nodes.ip).split(".")[:-1]) + "."
        preseeded = preseeded.replace(b"{{IP-PREFIX}}", node_cidr_prefix.encode())

        node_cidr_gateway = cidr_nodes.gateway()
        preseeded = preseeded.replace(b"{{GATEWAY}}", str(node_cidr_gateway).encode())

        node_cidr_netmask = cidr_nodes.netmask()
        preseeded = preseeded.replace(b"{{NETMASK}}", str(node_cidr_netmask).encode())

        preseeded = preseeded.replace(b"{{NAMESERVERS}}", " ".join(str(server_ip) for server_ip in upstream_dns_servers).encode())
        util.writefile(os.path.join(d, "preseed.cfg"), preseeded)

        inclusion += ["sshd_config.new", "preseed.cfg"]

        for package_name, (short_filename, package_bytes) in packages.verified_download_full(PACKAGES).items():
            assert "/" not in short_filename, "invalid package name: %s for %s" % (short_filename, package_name)
            assert short_filename.startswith(package_name + "_"), "invalid package name: %s for %s" % (short_filename, package_name)
            assert short_filename.endswith("_amd64.deb"), "invalid package name: %s for %s" % (short_filename, package_name)
            util.writefile(os.path.join(d, short_filename), package_bytes)
            inclusion.append(short_filename)

        cddir = os.path.join(d, "cd")
        os.mkdir(cddir)
        subprocess.check_call(["bsdtar", "-C", cddir, "-xzf", "/usr/share/homeworld/debian.iso"])
        subprocess.check_call(["chmod", "+w", "--recursive", cddir])

        if mode is not None:
            if mode not in MODES:
                command.fail("no such ISO mode: %s" % mode)
            MODES[mode](d, cddir, inclusion)

        with gzip.open(os.path.join(cddir, "initrd.gz"), "ab") as f:
            f.write(subprocess.check_output(["cpio", "--create", "--format=newc"],
                           input="".join("%s\n" % filename for filename in inclusion).encode(), cwd=d))

        files_for_md5sum = subprocess.check_output(["find", ".", "-follow", "-type", "f", "-print0"], cwd=cddir).decode().split("\0")
        assert files_for_md5sum.pop() == ""
        md5s = subprocess.check_output(["md5sum", "--"] + files_for_md5sum, cwd=cddir)
        util.writefile(os.path.join(cddir, "md5sum.txt"), md5s)

        temp_iso = os.path.join(d, "temp.iso")
        subprocess.check_call(["xorriso", "-as", "mkisofs", "-quiet", "-o", temp_iso, "-r", "-J", "-c", "boot.cat", "-b", "isolinux.bin", "-no-emul-boot", "-boot-load-size", "4", "-boot-info-table", cddir])
        subprocess.check_call(["isohybrid", "-h", "64", "-s", "32", temp_iso])
        util.copy(temp_iso, iso_image)


main_command = command.mux_map("commands about building installation ISOs", {
    "gen": command.wrap("generate ISO", gen_iso),
    "passphrases": command.wrap("decrypt a list of passphrases used by recently-generated ISOs", list_passphrases),
})
