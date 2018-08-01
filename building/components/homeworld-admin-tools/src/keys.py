import os
import subprocess
import tempfile

import command
import configuration
import keycrypt


def import_keytab(node, keytab_file):
    if not configuration.get_config().has_node(node):
        command.fail("no such node: %s" % node)
    keytab_target = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node)
    keycrypt.gpg_encrypt_file(keytab_file, keytab_target)


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


def rotate_keytab(node):
    return keytab_op(node, "rotate")


def delold_keytab(node):
    return keytab_op(node, "delold")


def list_keytabs(keytab=None):
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


def export_keytab(node, keytab_file):
    keytab_source = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node)
    if not os.path.exists(keytab_source):
        command.fail("no keytab for node %s" % node)
    keycrypt.gpg_decrypt_file(keytab_source, keytab_file)


keytab_command = command.mux_map("commands about keytabs granted by external sources", {
    "import": command.wrap("import and encrypt a keytab for a particular server", import_keytab),
    "rotate": command.wrap("decrypt, rotate, and re-encrypt the keytab for a particular server", rotate_keytab),
    "delold": command.wrap("decrypt, delete old entries from, and re-encrypt a keytab", delold_keytab),
    "list": command.wrap("decrypt and list one or all of the stored keytabs", list_keytabs),
    "export": command.wrap("decrypt and export the keytab for a particular server", export_keytab),
})
