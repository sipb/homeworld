import os
import subprocess
import tempfile

import authority
import command
import configuration
import keycrypt
import util


@command.wrap
def import_keytab(node, keytab_file):
    "import and encrypt a keytab for a particular server"

    if not configuration.get_config().has_node(node):
        command.fail("no such node: %s" % node)
    keytab_target = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node)
    keycrypt.gpg_encrypt_file(keytab_file, keytab_target)


def check_pem_type(filepath, expect):
    with open(filepath, "r") as f:
        first_line = f.readline()
    if not first_line.startswith("-----BEGIN ") or not first_line.rstrip().endswith("-----"):
        command.fail("not a PEM file: %s" % filepath)
    pem_header_type = first_line.rstrip()[len("-----BEGIN "):-len("-----")]
    if pem_header_type != expect:
        command.fail("incorrect PEM header: expected %s, not %s" % (expect, pem_header_type))


@command.wrap
def import_https(name, keyfile, certfile):
    "import and encrypt a HTTPS keypair for a particular server"
    check_pem_type(certfile, "CERTIFICATE")
    check_pem_type(keyfile, "RSA PRIVATE KEY")

    keypath = os.path.join(configuration.get_project(), "https.%s.key.crypt" % name)
    certpath = os.path.join(configuration.get_project(), "https.%s.pem" % name)

    keycrypt.gpg_encrypt_file(keyfile, keypath)
    util.copy(certfile, certpath)


def decrypt_https(hostname):
    return keycrypt.gpg_decrypt_to_memory(os.path.join(configuration.get_project(), "https.%s.key.crypt" % hostname)), \
           util.readfile(os.path.join(configuration.get_project(), "https.%s.pem" % hostname))


def keytab_op(node, op):
    if not configuration.get_config().has_node(node):
        command.fail("no such node: %s" % node)
    keytab_source = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node)
    keytab_target = os.path.join(configuration.get_project(), "keytab.%s.crypt.tmp" % node)
    with tempfile.TemporaryDirectory() as d:
        keytab_temp = os.path.join(d, "keytab.temp")
        keycrypt.gpg_decrypt_file(keytab_source, keytab_temp)
        if op == "rotate":
            operation = ["k5srvutil", "-f", keytab_temp, "change", "-e", "aes256-cts:normal,aes128-cts:normal"]
        elif op == "delold":
            operation = ["k5srvutil", "-f", keytab_temp, "delold"]
        else:
            command.fail("internal error: no such operation %s" % op)
        subprocess.check_call(operation)
        keycrypt.gpg_encrypt_file(keytab_temp, keytab_target)
    os.remove(keytab_source)
    os.rename(keytab_target, keytab_source)


@command.wrap
def rotate_keytab(node):
    "decrypt, rotate, and re-encrypt the keytab for a particular server"
    return keytab_op(node, "rotate")


@command.wrap
def delold_keytab(node):
    "decrypt, delete old entries from, and re-encrypt a keytab"
    return keytab_op(node, "delold")


@command.wrap
def list_keytabs(keytab=None):
    "decrypt and list one or all of the stored keytabs"
    keytabs = [".".join(kt.split(".")[1:-1]) for kt in os.listdir(configuration.get_project())
               if kt.startswith("keytab.") and kt.endswith(".crypt")]
    if keytab is not None:
        if keytab not in keytabs:
            command.fail("no keytab found for: %s" % keytab)
        keytabs = [keytab]
    with tempfile.TemporaryDirectory() as d:
        keytab_dest = os.path.join(d, "keytab.decrypt")
        for kt in keytabs:
            keytab_source = os.path.join(configuration.get_project(), "keytab.%s.crypt" % kt)
            keycrypt.gpg_decrypt_file(keytab_source, keytab_dest)
            print("== listing for %s ==" % kt)
            subprocess.check_call(["k5srvutil", "-f", keytab_dest, "list"])
            os.remove(keytab_dest)


@command.wrap
def export_keytab(node, keytab_file):
    "decrypt and export the keytab for a particular server"
    keytab_source = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node)
    if not os.path.exists(keytab_source):
        command.fail("no keytab for node %s" % node)
    keycrypt.gpg_decrypt_file(keytab_source, keytab_file)


@command.wrap
def export_https(name, keyout, certout):
    "decrypt and export the HTTPS keypair for a particular server"
    keypath = os.path.join(configuration.get_project(), "https.%s.key.crypt" % name)
    certpath = os.path.join(configuration.get_project(), "https.%s.pem" % name)

    keycrypt.gpg_decrypt_file(keypath, keyout)
    util.copy(certpath, certout)


@command.wrap
def gen_local_https_cert(name):
    "generate and encrypt a HTTPS keypair using the cluster-internal CA"
    if "," in name:
        command.fail("cannot create https cert with comma: would be misinterpreted by keylocalcert")
    print("generating local-only https cert for", name, "via local bypass method")
    with tempfile.TemporaryDirectory() as dir:
        ca_key = os.path.join(dir, "ca.key")
        ca_pem = os.path.join(dir, "ca.pem")
        key_path = os.path.join(dir, "gen.key")
        cert_path = os.path.join(dir, "gen.pem")
        util.writefile(ca_key, authority.get_decrypted_by_filename("./clusterca.key"))
        pem = authority.get_pubkey_by_filename("./clusterca.pem")
        util.writefile(ca_pem, pem)
        os.chmod(ca_key, 0o600)
        subprocess.check_call(["keylocalcert", ca_key, ca_pem, name, "4h", key_path, cert_path, name, ""])
        import_https(name, key_path, cert_path)
    print("generated local-only https cert!")


keytab_command = command.Mux("commands about keytabs granted by external sources", {
    "import": import_keytab,
    "rotate": rotate_keytab,
    "delold": delold_keytab,
    "list": list_keytabs,
    "export": export_keytab,
})

https_command = command.Mux("commands about HTTPS certs granted by external sources", {
    "import": import_https,
    "export": export_https,
    "genlocal": gen_local_https_cert,
})
