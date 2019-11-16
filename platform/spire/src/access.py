from typing import Tuple
import base64
import binascii
import os
import subprocess
import tempfile
import time
import traceback

import authority
import command
import configuration
import setup
import util


DEFAULT_ROTATE_INTERVAL = 60 * 60 * 2  # rotate local key every two hours (if we happen to renew)


def needs_rotate(path, interval=DEFAULT_ROTATE_INTERVAL):
    try:
        result = os.stat(path)
        time_since_last_rotate = time.time() - result.st_mtime
        return time_since_last_rotate >= interval
    except FileNotFoundError:
        return True


KEYREQ_ERROR_CODES = {
    1: "ERR_UNKNOWN_FAILURE",
    2: "ERR_CANNOT_ESTABLISH_CONNECTION",
    3: "ERR_NO_ACCESS",
    254: "ERR_INVALID_CONFIG",
    255: "ERR_INVALID_INVOCATION",
}

KNC_STDERR_START_TAG = "--- knc stderr start ---"
KNC_STDERR_END_TAG = "--- knc stderr end ---"


def diagnose_keyreq_error(errcode: int, err: str) -> Tuple[str, str]:
    if errcode not in KEYREQ_ERROR_CODES:
        return "unknown error code {}".format(errcode), None

    error_code_meaning = KEYREQ_ERROR_CODES[errcode]

    if errcode == 2:
        knc_stderr_start = err.find(KNC_STDERR_START_TAG)
        knc_stderr_end = err.find(KNC_STDERR_END_TAG)
        if knc_stderr_start != -1 and knc_stderr_end != -1:
            knc_stderr = err[knc_stderr_start + len(KNC_STDERR_START_TAG):knc_stderr_end]
            if "gstd_initiate: continuation failed" in knc_stderr:
                return error_code_meaning, "the server's keygateway might be broken."
            elif "gss_init_sec_context: No Kerberos credentials available" in knc_stderr or "gstd_error: gss_init_sec_context: Ticket expired" in knc_stderr:
                return error_code_meaning, "do you have valid kerberos tickets?"
        if "empty response, likely because the server does not recognize your Kerberos identity" in err:
            return error_code_meaning, "your kerberos tickets might be for the wrong instance."

    return error_code_meaning, None

class KeyreqFailed(command.CommandFailedException):
    def __init__(self, returncode, err):
        error_code_meaning, fail_hint = diagnose_keyreq_error(returncode, err)
        super().__init__("keyreq failed: {}".format(error_code_meaning), fail_hint)


def call_keyreq(keyreq_command, *params):
    config = configuration.get_config()
    keyserver_domain = config.keyserver.hostname + "." + config.external_domain + ":20557"

    with tempfile.TemporaryDirectory() as tdir:
        https_cert_path = os.path.join(tdir, "clusterca.pem")
        util.writefile(https_cert_path, authority.get_pubkey_by_filename("./clusterca.pem"))
        keyreq_sp = subprocess.Popen(["keyreq", keyreq_command, https_cert_path, keyserver_domain] + list(params), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        output, err_bytes = keyreq_sp.communicate()
        if keyreq_sp.returncode != 0:
            err = err_bytes.decode()
            raise KeyreqFailed(keyreq_sp.returncode, err)
        return output


def renew_ssh_cert() -> str:
    project_dir = configuration.get_project()
    keypath = os.path.join(project_dir, "ssh-key")
    # TODO: pull name of authority from worldconfig/apis.go
    refresh_cert(keypath, keypath + "-cert.pub", None, "ssh", "ssh-user", "ssh-user.pub")
    return keypath


def refresh_cert(key_path, cert_path, ca_path, variant, ca_key_name, ca_cert_name):
    errs = []

    try:
        if configuration.get_config().is_kerberos_enabled():
            print("rotating", variant, "certs via keyreq")
            if ca_path is None:
                call_keyreq(variant + "-cert", key_path, cert_path)
            else:
                call_keyreq(variant + "-cert", key_path, cert_path, ca_path)
            return
    except Exception as e:
        print("[keyreq failed, set SPIRE_DEBUG for traceback]")
        if os.environ.get('SPIRE_DEBUG'):
            traceback.print_exc()
        errs.append(e)

    try:
        print("generating", variant, "cert via local bypass method")
        with tempfile.TemporaryDirectory() as dir:
            ca_key = os.path.join(dir, ca_key_name)
            ca_pem = os.path.join(dir, ca_cert_name)
            util.writefile(ca_key, authority.get_decrypted_by_filename("./" + ca_key_name))
            pem = authority.get_pubkey_by_filename("./" + ca_cert_name)
            if ca_path is not None:
                util.writefile(ca_path, pem)
            util.writefile(ca_pem, pem)
            os.chmod(ca_key, 0o600)
            if variant == "kube":
                name = "root:direct"
                orgs = ["system:masters"]
            else:
                name = "temporary-%s-bypass-grant" % variant
                orgs = []
            subprocess.check_call(["keylocalcert", ca_key, ca_pem, name, "4h", key_path, cert_path, "", ",".join(orgs)])
        return
    except Exception as e:
        print("[local bypass failed, set SPIRE_DEBUG for traceback]")
        if os.environ.get('SPIRE_DEBUG'):
            traceback.print_exc()
        errs.append(e)

    if len(errs) > 1:
        raise command.MultipleExceptions('refresh_cert failed', errs)
    raise Exception('refresh_cert failed') from errs[0]


@command.wrap
def access_ssh(no_add_to_agent: bool=False):
    """
    request SSH access to the cluster

    no_add_to_agent: do not add the resulting ssh key to ssh-agent
    """
    keypath = renew_ssh_cert()
    print("===== v CERTIFICATE DETAILS v =====")
    subprocess.check_call(["ssh-keygen", "-L", "-f", keypath + "-cert.pub"])
    print("===== ^ CERTIFICATE DETAILS ^ =====")
    if not no_add_to_agent:
        # TODO: clear old identities
        try:
            ssh_add_output = subprocess.check_output(["ssh-add", "--", keypath], stderr=subprocess.STDOUT)
            # if the user is using gnome, gnome-keyring might
            # masquerade as ssh-agent and provide a zero exit
            # code despite failing to add the certificate
            if b"add failed" in ssh_add_output:
                fail_hint = "do you have an ssh-agent?\n" \
                    "(gnome-keyring does not count)"
                command.fail("*** ssh-add failed! ***", fail_hint)
        except subprocess.CalledProcessError:
            fail_hint = "ssh-add returned non-zero exit code. do you have an ssh-agent?"
            command.fail("*** ssh-add failed! ***", fail_hint)


HOMEWORLD_KNOWN_HOSTS_MARKER = "homeworld-keydef"


def _is_homeworld_keydef_line(line):
    return line.startswith("@cert-authority ") and line.endswith(" " + HOMEWORLD_KNOWN_HOSTS_MARKER)


def _replace_cert_authority(known_hosts_lines: list, machine_list: str, pubkey: bytes) -> list:
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
    # machine_list is trusted and locally-generated, so no validation is necessary
    rebuilt.append("@cert-authority %s ssh-rsa %s %s"
                   % (machine_list, base64.b64encode(b64data).decode(), HOMEWORLD_KNOWN_HOSTS_MARKER))
    return rebuilt


def get_known_hosts_path() -> str:
    homedir = os.getenv("HOME")
    if homedir is None:
        command.fail("could not determine home directory, so could not find ~/.ssh/known_hosts")
    return os.path.join(homedir, ".ssh", "known_hosts")


@command.wrap
def update_known_hosts():
    "update ~/.ssh/known_hosts file with @ca-certificates directive"
    config = configuration.Config.load_from_project()
    machines = ",".join("%s.%s" % (node.hostname, config.external_domain) for node in config.nodes)
    cert_authority_pubkey = authority.get_pubkey_by_filename("./ssh-host.pub")
    known_hosts_path = get_known_hosts_path()
    known_hosts_old = util.readfile(known_hosts_path).decode().split("\n") if os.path.exists(known_hosts_path) else []

    if known_hosts_old and not known_hosts_old[-1]:
        known_hosts_old.pop()

    known_hosts_new = _replace_cert_authority(known_hosts_old, machines, cert_authority_pubkey)

    util.writefile(known_hosts_path, ("\n".join(known_hosts_new) + "\n").encode())
    print("~/.ssh/known_hosts updated")


def call_etcdctl(params: list, return_result: bool):
    project_dir = configuration.get_project()
    endpoints = configuration.get_etcd_endpoints()

    etcd_key_path = os.path.join(project_dir, "etcd-access.key")
    etcd_cert_path = os.path.join(project_dir, "etcd-access.pem")
    etcd_ca_path = os.path.join(project_dir, "etcd-ca.pem")
    if needs_rotate(etcd_cert_path):
        refresh_cert(etcd_key_path, etcd_cert_path, etcd_ca_path, "etcd", "etcd-client.key", "etcd-client.pem")

    args = ["etcdctl", "--cert-file", etcd_cert_path, "--key-file", etcd_key_path,
                       "--ca-file", etcd_ca_path, "--endpoints", endpoints] + list(params)

    if return_result:
        return subprocess.check_output(args)
    else:
        subprocess.check_call(args)


@command.wrap
def dispatch_etcdctl(*params: str):
    "invoke commands through the etcdctl wrapper"
    call_etcdctl(params, False)


def call_kubectl(params, return_result: bool):
    kubeconfig_data = configuration.get_local_kubeconfig()
    key_path, cert_path, ca_path = configuration.get_kube_cert_paths()

    if needs_rotate(cert_path):
        refresh_cert(key_path, cert_path, ca_path, "kube", "kubernetes.key", "kubernetes.pem")

    with tempfile.TemporaryDirectory() as f:
        kubeconfig_path = os.path.join(f, "temp-kubeconfig")
        util.writefile(kubeconfig_path, kubeconfig_data.encode())
        args = ["hyperkube", "kubectl", "--kubeconfig", kubeconfig_path] + list(params)
        if return_result:
            return subprocess.check_output(args)
        else:
            subprocess.check_call(args)


@command.wrap
def dispatch_kubectl(*params: str):
    "invoke commands through the kubectl wrapper"
    call_kubectl(params, False)


@command.wrapop
def ssh_foreach(ops: command.Operations, node_kind: str, *params: str):
    "invoke commands on every node (or every node of a given kind) in the cluster"
    config = configuration.get_config()
    valid_node_kinds = configuration.Node.VALID_NODE_KINDS
    if not (node_kind == "node" or node_kind == "kube" or node_kind in valid_node_kinds):
        command.fail("usage: spire foreach {node,kube," + ",".join(valid_node_kinds) + "} command")
    for node in config.nodes:
        if node_kind == "node" or node.kind == node_kind or (node_kind == "kube" and node.kind != "supervisor"):
            setup.ssh_cmd(ops, "run command on @HOST", node, *params)


def compute_fingerprint(key: str) -> str:
    text = subprocess.check_output(["ssh-keygen", "-l", "-f", "-"], input=key.encode(), stderr=subprocess.STDOUT).decode()
    if not text.endswith("\n") or text.count("\n") != 1:
        raise Exception("invalid format of result from ssh-keygen -l: expected exactly one line")
    return text.rstrip("\n")


def hostkeys_by_fingerprint(node: configuration.Node, fingerprints: list):
    keys = []
    for line in subprocess.check_output(["ssh-keyscan", "-T", "1", "--", str(node.ip)]).decode().split("\n"):
        if not line or line.startswith("#"): continue
        if line.count(" ") < 1:
            raise Exception("invalid format of result from ssh-keyscan: expected two fields")
        server, key = line.split(" ", 1)
        if server != str(node.ip):
            raise Exception("ssh-keyscan returned server information for a different server than expected")
        fingerprint = compute_fingerprint(key)
        if fingerprint in fingerprints:
            keys.append(key)
    return keys


def pull_supervisor_key(fingerprints):
    config = configuration.get_config()
    node = config.keyserver
    known_hosts = get_known_hosts_path()

    for remove in ["%s.%s" % (node.hostname, config.external_domain), str(node.ip)]:
        subprocess.check_call(["ssh-keygen", "-f", known_hosts, "-R", remove],
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    keys = hostkeys_by_fingerprint(node, fingerprints)
    with open(known_hosts, "a") as f:
        for key in keys:
            f.write("%s.%s %s\n" % (node.hostname, config.external_domain, key))


@command.wrap
def pull_supervisor_key_from(source_file):
    "update ~/.ssh/known_hosts file with the supervisor host keys, based on their known hashes"
    pull_supervisor_key(util.readfile(source_file).decode().strip().split("\n"))


etcdctl_command = dispatch_etcdctl
kubectl_command = dispatch_kubectl
foreach_command = ssh_foreach
main_command = command.Mux("commands about establishing access to a cluster", {
    "ssh": access_ssh,

    "update-known-hosts": update_known_hosts,
    "pull-supervisor-key": pull_supervisor_key_from,
})
