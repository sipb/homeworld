import configuration
import subprocess

SSH_BASE = ["ssh", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=1"]
SCP_BASE = ["scp", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=1"]


def ssh_get_login(node: configuration.Node) -> str:  # returns root@<HOSTNAME>.<EXTERNAL_DOMAIN>
    config = configuration.get_config()
    return "root@%s.%s" % (node.hostname, config.external_domain)


def build_ssh(node: configuration.Node, *script: str):
    return SSH_BASE + [ssh_get_login(node), "--"] + list(script)


def build_scp_up(node: configuration.Node, source_path: str, dest_path: str):
    return SCP_BASE + ["--", source_path, ssh_get_login(node) + ":" + dest_path]


def check_ssh(node: configuration.Node, *script: str):
    subprocess.check_call(build_ssh(node, *script))


def check_ssh_output(node: configuration.Node, *script: str):
    return subprocess.check_output(build_ssh(node, *script))


def check_scp_up(node: configuration.Node, source_path: str, dest_path: str):
    subprocess.check_call(build_scp_up(node, source_path, dest_path))
