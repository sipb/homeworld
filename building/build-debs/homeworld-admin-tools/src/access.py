import time
import os
import command
import subprocess
import configuration
import tempfile
import authority
import util
import base64
import binascii

DEFAULT_ROTATE_INTERVAL = 60 * 60 * 2  # rotate local key every two hours (if we happen to renew)
DEFAULT_SHORTLIVED_RSA_BITS = 2048


def needs_rotate(path, interval=DEFAULT_ROTATE_INTERVAL):
    try:
        result = os.stat(path)
        time_since_last_rotate = time.time() - result.st_mtime
        return time_since_last_rotate >= interval
    except FileNotFoundError:
        return True


def create_or_rotate_custom_ssh_key(interval=DEFAULT_ROTATE_INTERVAL, bits=DEFAULT_SHORTLIVED_RSA_BITS):
    project_dir = configuration.get_project()
    keypath = os.path.join(project_dir, "ssh-key")
    if needs_rotate(keypath, interval):
        if os.path.exists(keypath):
            os.remove(keypath)
        if os.path.exists(keypath + ".pub"):
            os.remove(keypath + ".pub")
        if os.path.exists(keypath + "-cert.pub"):
            os.remove(keypath + "-cert.pub")
        # 2048 bits is sufficient for a key only used for the duration of the certificate (probably four hours)
        subprocess.check_call(["ssh-keygen",
                               "-f", keypath,                 # output file
                               "-t", "rsa", "-b", str(bits),  # a <bits>-bit RSA key
                               "-C", "autogen-homeworld",     # generic comment
                               "-N", ""])                     # no passphrase
    return keypath


def call_keyreq(command, *params, collect=False):
    config = configuration.Config.load_from_project()
    keyserver_domain = config.keyserver.hostname + "." + config.external_domain + ":20557"

    invoke_variant = subprocess.check_output if collect else subprocess.check_call

    with tempfile.TemporaryDirectory() as tdir:
        https_cert_path = os.path.join(tdir, "server.pem")
        util.writefile(https_cert_path, authority.get_key_by_filename("./server.pem"))
        return invoke_variant(["keyreq", command, https_cert_path, keyserver_domain] + list(params))


def renew_ssh_cert() -> str:
    keypath = create_or_rotate_custom_ssh_key()
    call_keyreq("ssh-cert", keypath + ".pub", keypath + "-cert.pub")
    return keypath


def access_ssh(add_to_agent=False):
    keypath = renew_ssh_cert()
    print("===== v CERTIFICATE DETAILS v =====")
    subprocess.check_call(["ssh-keygen", "-L", "-f", keypath + "-cert.pub"])
    print("===== ^ CERTIFICATE DETAILS ^ =====")
    if add_to_agent:
        # TODO: clear old identities
        subprocess.check_call(["ssh-add", "--", keypath])


def access_ssh_with_add():
    access_ssh(add_to_agent=True)


HOMEWORLD_KNOWN_HOSTS_MARKER = "homeworld-keydef"


def _is_homeworld_keydef_line(line):
    return line.startswith("@cert-authority ") and line.endswith(" " + HOMEWORLD_KNOWN_HOSTS_MARKER)


def _replace_cert_authority(known_hosts_lines: list, machine_list: str, pubkey: bytes) -> list:
    # unlike the original golang implementation, machine_list is trusted and locally-generated, so no validation is
    # necessary.
    pubkey_parts = pubkey.split(b" ")

    if len(pubkey_parts) != 2:
        command.fail("invalid CA pubkey while parsing certificate authority")
    if pubkey_parts[0] != b"ssh-rsa":
        command.fail("unexpected CA type (%s instead of ssh-rsa) while parsing certificate authority" % pubkey_parts[0])
    try:
        b64data = base64.b64decode(pubkey_parts[1], validate=True)
    except binascii.Error as e:
        command.fail("invalid base64-encoded pubkey: %s" % e)

    rebuilt = [line for line in known_hosts_lines if not _is_homeworld_keydef_line(line)]
    rebuilt.append("@cert-authority %s ssh-rsa %s %s"
                   % (machine_list, base64.b64encode(b64data).decode(), HOMEWORLD_KNOWN_HOSTS_MARKER))
    return rebuilt


def update_known_hosts():
    # uses local copies of machine list and ssh-host pubkey
    # TODO: eliminate now-redundant machine.list download from keyserver
    machines = configuration.get_machine_list_file().strip()
    cert_authority_pubkey = authority.get_key_by_filename("./ssh_host_ca.pub")
    homedir = os.getenv("HOME")
    if homedir is None:
        command.fail("could not determine home directory, so could not find ~/.ssh/known_hosts")
    known_hosts_path = os.path.join(homedir, ".ssh", "known_hosts")
    known_hosts_old = util.readfile(known_hosts_path).decode().split("\n") if os.path.exists(known_hosts_path) else []

    if known_hosts_old and not known_hosts_old[-1]:
        known_hosts_old.pop()

    known_hosts_new = _replace_cert_authority(known_hosts_old, machines, cert_authority_pubkey)

    util.writefile(known_hosts_path, ("\n".join(known_hosts_new) + "\n").encode())
    print("~/.ssh/known_hosts updated")


def get_kube_cert_paths():
    project_dir = configuration.get_project()
    return os.path.join(project_dir, "kube-access.key"),\
           os.path.join(project_dir, "kube-access.pem"),\
           os.path.join(project_dir, "kube-ca.pem")


def dispatch_etcdctl(*params):
    project_dir = configuration.get_project()
    endpoints = configuration.get_etcd_endpoints()

    etcd_key_path = os.path.join(project_dir, "etcd-access.key")
    etcd_cert_path = os.path.join(project_dir, "etcd-access.pem")
    etcd_ca_path = os.path.join(project_dir, "etcd-ca.pem")
    if needs_rotate(etcd_cert_path):
        print("rotating etcd certs...")
        call_keyreq("etcd-cert", etcd_key_path, etcd_cert_path, etcd_ca_path)

    subprocess.check_call(["etcdctl", "--cert-file", etcd_cert_path, "--key-file", etcd_key_path,
                           "--ca-file", etcd_ca_path, "--endpoints", endpoints] + list(params))


def dispatch_kubectl(*params):
    kubeconfig_data = configuration.get_local_kubeconfig()
    key_path, cert_path, ca_path = get_kube_cert_paths()

    if needs_rotate(cert_path):
        print("rotating kubernetes certs...")
        call_keyreq("kube-cert", key_path, cert_path, ca_path)

    with tempfile.TemporaryDirectory() as f:
        kubeconfig_path = os.path.join(f, "temp-kubeconfig")
        util.writefile(kubeconfig_path, kubeconfig_data.encode())
        subprocess.check_call(["hyperkube", "kubectl", "--kubeconfig", kubeconfig_path] + list(params))


etcdctl_command = command.wrap("invoke commands through the etcdctl wrapper", dispatch_etcdctl)
kubectl_command = command.wrap("invoke commands through the kubectl wrapper", dispatch_kubectl)
main_command = command.mux_map("commands about establishing access to a cluster", {
    "ssh": command.wrap("request SSH access to the cluster and add it to the SSH agent", access_ssh_with_add),
    "ssh-fetch": command.wrap("request SSH access to the cluster but do not register it with the agent", access_ssh),
    "update-known-hosts": command.wrap("update ~/.ssh/known_hosts file with @ca-certificates directive", update_known_hosts)
})
