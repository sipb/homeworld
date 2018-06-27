import authority
import command
import datetime

import configuration
import os
import tempfile
import subprocess
import shutil
import util
import keycrypt
import setup
from version import get_git_version


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
    for passfile in sorted(os.listdir(passwords)):
        if passfile.startswith("at-") and passfile.endswith(".gpg"):
            date = passfile[3:-4]
            passph = keycrypt.gpg_decrypt_to_memory(os.path.join(passwords, passfile)).decode()
            print("   ", date, "=>", passph)
    print("End of list.")


# TODO: autodownload image_input
def gen_iso(iso_image, authorized_key, image_input):
    with tempfile.TemporaryDirectory() as d:
        subprocess.check_call(["tar", "-xf", image_input, "-C", d])

        inclusion = []

        with open(os.path.join(d, "dns_bootstrap_lines"), "w") as outfile:
            outfile.writelines(setup.dns_bootstrap_lines())

        inclusion += ["dns_bootstrap_lines"]
        util.copy(authorized_key, os.path.join(d, "authorized.pub"))
        util.writefile(os.path.join(d, "keyservertls.pem"), authority.get_pubkey_by_filename("./server.pem"))
        inclusion += ["authorized.pub", "keyservertls.pem"]

        for variant in configuration.KEYCLIENT_VARIANTS:
            util.writefile(os.path.join(d, "keyclient-%s.yaml" % variant), configuration.get_keyclient_yaml(variant).encode())
            inclusion.append("keyclient-%s.yaml" % variant)

        generated_password = util.pwgen(20)
        creation_time = datetime.datetime.now().isoformat()
        git_hash = get_git_version().encode()
        add_password_to_log(generated_password, creation_time)
        print("generated password added to log")

        settings = {  # TODO: don't hardcode these
            "DNS_SERVERS": "18.70.0.160 18.71.0.151 18.72.0.3",
            "ADDRESS_PREFIX": "18.4.60.",
            "ADDRESS_SUFFIX": "/23",
            "GATEWAY": "18.4.60.1",
            "PASSWORD": generated_password.decode(),  # TODO: use util.mkpasswd(generated_password) instead
            "BUILDDATE": creation_time,
            "GIT_HASH": git_hash.decode(),
        }
        settings_text = "".join('%s="%s"\n' % (k, v.replace('"', '"\'"\'"')) for k, v in settings.items())
        util.writefile(os.path.join(d, "settings"), settings_text.encode())

        # TODO: don't have this remote-code-execution opening
        subprocess.check_call(["./finalize.sh", os.path.abspath(iso_image)], cwd=d)


main_command = command.mux_map("commands about building installation ISOs", {
    "gen": command.wrap("generate ISO", gen_iso),
    "passphrases": command.wrap("decrypt a list of passphrases used by recently-generated ISOs", list_passphrases),
})
