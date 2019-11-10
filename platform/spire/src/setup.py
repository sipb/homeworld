import os
import subprocess

import authority
import command
import configuration
import keycrypt
import resource
import ssh


def escape_shell(param: str) -> str:
    # replaces ' -> '"'"'
    return "'" + param.replace("'", "'\"'\"'") + "'"


def ssh_raw(ops, name: str, node: configuration.Node, script: str, in_directory: str=None, redirect_to: str=None)\
        -> None:
    if redirect_to:
        script = "(%s) >%s" % (script, escape_shell(redirect_to))
    if in_directory:
        script = "cd %s && %s" % (escape_shell(in_directory), script)
    ops.add_operation(name.replace('@HOST', node.hostname),
                      lambda: ssh.check_ssh(node, script))

def ssh_cmd(ops, name: str, node: configuration.Node, *argv: str, in_directory: str=None, redirect_to: str=None)\
        -> None:
    ssh_raw(ops, name, node, " ".join(escape_shell(param) for param in argv),
            in_directory=in_directory, redirect_to=redirect_to)

def ssh_mkdir(ops, name: str, node: configuration.Node, *paths: str, with_parents: bool=True) -> None:
    options = ["-p"] if with_parents else []
    ssh_cmd(ops, name, node, "mkdir", *(options + ["--"] + list(paths)))

def ssh_upload_path(ops, name: str, node: configuration.Node, source_path: str, dest_path: str) -> None:
    ops.add_operation(name.replace('@HOST', node.hostname),
                      lambda: ssh.check_scp_up(node, source_path, dest_path))

def ssh_upload_bytes(ops, name: str, node: configuration.Node, source_bytes: bytes, dest_path: str) -> None:
    ops.add_operation(name.replace('@HOST', node.hostname),
                      lambda: ssh.upload_bytes(node, source_bytes, dest_path))


AUTHORITY_DIR = "/etc/homeworld/keyserver/authorities"
STATICS_DIR = "/etc/homeworld/keyserver/static"
CONFIG_DIR = "/etc/homeworld/config"
KEYCLIENT_DIR = "/etc/homeworld/keyclient"
KEYTAB_PATH = "/etc/krb5.keytab"


@command.wrapop
def setup_keyserver(ops: command.Operations) -> None:
    "deploy keys and configuration for keyserver; start keyserver"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_mkdir(ops, "create directories on @HOST", node, AUTHORITY_DIR, STATICS_DIR, CONFIG_DIR)
        for name, data in authority.iterate_keys_decrypted():
            # TODO: keep these keys in memory
            if "/" in name:
                command.fail("found key in upload list with invalid filename")
            # TODO: avoid keeping these keys in memory for this long
            ssh_upload_bytes(ops, "upload authority %s to @HOST" % name, node, data, os.path.join(AUTHORITY_DIR, name))
        ssh_upload_bytes(ops, "upload cluster config to @HOST", node,
                         configuration.get_cluster_conf().encode(), STATICS_DIR + "/cluster.conf")
        ssh_upload_path(ops, "upload cluster setup to @HOST", node,
                        configuration.Config.get_setup_path(), CONFIG_DIR + "/setup.yaml")
        ssh_cmd(ops, "enable keyserver on @HOST", node, "systemctl", "enable", "keyserver.service")
        ssh_cmd(ops, "start keyserver on @HOST", node, "systemctl", "restart", "keyserver.service")

def redeploy_keyserver(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        # delete the existing configs
        ops.ssh_rm("delete existing cluster config from @HOST", node, STATICS_DIR + "/cluster.conf")
        ops.ssh_rm("delete existing keyserver config from @HOST", node, CONFIG_DIR + "/setup.yaml")
        # redeploy new config
        ops.ssh_upload_bytes("reupload cluster config to @HOST", node,
            configuration.get_cluster_conf().encode(), STATICS_DIR + "/cluster.conf")
        ops.ssh_upload_path("upload cluster setup to @HOST", node,
                            configuration.Config.get_setup_path(), CONFIG_DIR + "/setup.yaml")
        # restart the keyserver
        ops.ssh("restart keyserver on @HOST", node, "systemctl", "restart", "keyserver.service")

def redeploy_keyclients(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        # delete existing cluster configuration from non-supervisor nodes
        if node.kind != "supervisor":
            ops.ssh_rm("delete existing cluster config from @HOST", node, CONFIG_DIR + "/cluster.conf")
        ops.ssh_rm("delete existing local config from @HOST", node, CONFIG_DIR + "/local.conf")
        # restart local keyclient (will regenerate configs on restart)
        ops.ssh("restart keyclient daemon on @HOST", node, "systemctl", "restart", "keyclient.service")

@command.wrapop
def admit_keyserver(ops: command.Operations) -> None:
    "admit the keyserver into the cluster during bootstrapping"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        domain = node.hostname + "." + config.external_domain
        ssh_cmd(ops, "request bootstrap token for @HOST", node,
                "keyinitadmit", domain,
                redirect_to=KEYCLIENT_DIR + "/bootstrap.token")
        # TODO: do we need to poke the keyclient to make sure it tries again?
        # TODO: don't wait four seconds if it isn't necessary
        ssh_cmd(ops, "kick keyclient daemon on @HOST", node, "systemctl", "restart", "keyclient")
        # if it doesn't exist, this command will fail.
        ssh_cmd(ops, "confirm that @HOST was admitted", node, "test", "-e", KEYCLIENT_DIR + "/granting.pem")
        ssh_cmd(ops, "enable auth-monitor daemon on @HOST", node, "systemctl", "enable", "auth-monitor")
        ssh_cmd(ops, "start auth-monitor daemon on @HOST", node, "systemctl", "restart", "auth-monitor")


def modify_keygateway(ops: command.Operations, overwrite_keytab: bool) -> None:
    config = configuration.get_config()
    if not config.is_kerberos_enabled():
        print("keygateway disabled; skipping")
        return
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        # keytab is stored encrypted in the configuration folder
        keytab = os.path.join(configuration.get_project(), "keytab.%s.crypt" % node.hostname)
        decrypted = keycrypt.gpg_decrypt_to_memory(keytab)
        def safe_upload_keytab(node=node):
            if not overwrite_keytab:
                try:
                    existing_keytab = ssh.check_ssh_output(node, "cat", KEYTAB_PATH)
                except subprocess.CalledProcessError as e_test:
                    # if there is no existing keytab, cat will fail with error code 1
                    if e_test.returncode != 1:
                        command.fail(e_test)
                    print("no existing keytab found, uploading local keytab")
                else:
                    if existing_keytab != decrypted:
                        command.fail("existing keytab does not match local keytab")
                    return # existing keytab matches local keytab, no action required
            ssh.upload_bytes(node, decrypted, KEYTAB_PATH)
        ops.add_operation("upload keytab for {}".format(node), safe_upload_keytab)
        ssh_cmd(ops, "enable keygateway on @HOST", node, "systemctl", "enable", "keygateway")
        ssh_cmd(ops, "restart keygateway on @HOST", node, "systemctl", "restart", "keygateway")


@command.wrapop
def setup_keygateway(ops: command.Operations) -> None:
    "deploy keytab and start keygateway"
    modify_keygateway(ops, False)


@command.wrapop
def update_keygateway(ops: command.Operations) -> None:
    "update keytab and restart keygateway"
    modify_keygateway(ops, True)


@command.wrapop
def setup_supervisor_ssh(ops: command.Operations) -> None:
    "configure supervisor SSH access"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_config = resource.get("//spire/resources:sshd_config")
        ssh_upload_bytes(ops, "upload new ssh configuration to @HOST", node, ssh_config, "/etc/ssh/sshd_config")
        ssh_cmd(ops, "reload ssh configuration on @HOST", node, "systemctl", "restart", "ssh")
        ssh_raw(ops, "shift aside old authorized_keys on @HOST", node,
                "if [ -f /root/.ssh/authorized_keys ]; then " +
                "mv /root/.ssh/authorized_keys " +
                "/root/original_authorized_keys; fi")


def modify_dns_bootstrap(ops: command.Operations, is_install: bool) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        strip_cmd = "grep -vF AUTO-HOMEWORLD-BOOTSTRAP /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts"
        ssh_raw(ops, "strip bootstrapped dns on @HOST", node, strip_cmd)
        if is_install:
            for hostname, ip in config.dns_bootstrap.items():
                new_hosts_line = "%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP" % (ip, hostname)
                strip_cmd = "echo %s >>/etc/hosts" % escape_shell(new_hosts_line)
                ssh_raw(ops, "bootstrap dns on @HOST: %s" % hostname, node, strip_cmd)


def modify_temporary_dns(node: configuration.Node, additional: dict) -> None:
    ssh.check_ssh(node, "grep -vF AUTO-TEMP-DNS /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts")
    for hostname, ip in additional.items():
        new_hosts_line = "%s\t%s # AUTO-TEMP-DNS" % (ip, hostname)
        ssh.check_ssh(node, "echo %s >>/etc/hosts" % escape_shell(new_hosts_line))


def dns_bootstrap_lines() -> str:
    config = configuration.get_config()
    dns_hosts = config.dns_bootstrap.copy()
    dns_hosts["homeworld.private"] = config.keyserver.ip
    for node in config.nodes:
        full_hostname = "%s.%s" % (node.hostname, config.external_domain)
        if node.hostname in dns_hosts:
            command.fail("redundant /etc/hosts entry: %s", node.hostname)
        if full_hostname in dns_hosts:
            command.fail("redundant /etc/hosts entry: %s", full_hostname)
        dns_hosts[node.hostname] = node.ip
        dns_hosts[full_hostname] = node.ip
    return "".join("%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP\n" % (ip, hostname) for hostname, ip in dns_hosts.items())


@command.wrapop
def setup_dns_bootstrap(ops: command.Operations) -> None:
    "switch cluster nodes into 'bootstrapped DNS' mode"

    modify_dns_bootstrap(ops, True)


@command.wrapop
def teardown_dns_bootstrap(ops: command.Operations) -> None:
    "switch cluster nodes out of 'bootstrapped DNS' mode"

    modify_dns_bootstrap(ops, False)


@command.wrapop
def setup_bootstrap_registry(ops: command.Operations) -> None:
    "bring up the bootstrap container registry on the supervisor nodes"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue

        ssh_cmd(ops, "enable docker-registry on @HOST", node, "systemctl", "enable", "docker-registry")
        ssh_cmd(ops, "restart docker-registry on @HOST", node, "systemctl", "restart", "docker-registry")

        ssh_cmd(ops, "unmask nginx on @HOST", node, "systemctl", "unmask", "nginx")
        ssh_cmd(ops, "enable nginx on @HOST", node, "systemctl", "enable", "nginx")
        ssh_cmd(ops, "restart nginx on @HOST", node, "systemctl", "restart", "nginx")


@command.wrapop
def update_registry(ops: command.Operations) -> None:
    "upload the latest container versions to the bootstrap container registry"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_cmd(ops, "update apt repositories on @HOST", node, "apt-get", "update")
        ssh_cmd(ops, "update package of OCIs on @HOST", node, "apt-get", "install", "-y", "homeworld-oci-pack")
        ssh_cmd(ops, "upgrade apt packages on @HOST", node, "apt-get", "upgrade", "-y")
        ssh_cmd(ops, "re-push OCIs to registry on @HOST", node, "/usr/lib/homeworld/push-ocis.sh")

@command.wrapop
def setup_prometheus(ops: command.Operations) -> None:
    "bring up the supervisor node prometheus instance"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_upload_bytes(ops, "upload prometheus config to @HOST", node, configuration.get_prometheus_yaml().encode(),
                         "/etc/prometheus.yaml")
        ssh_cmd(ops, "enable prometheus on @HOST", node, "systemctl", "enable", "prometheus")
        ssh_cmd(ops, "restart prometheus on @HOST", node, "systemctl", "restart", "prometheus")


main_command = command.Mux("commands about setting up a cluster", {
    "keyserver": setup_keyserver,
    "self-admit": admit_keyserver,
    "keygateway": setup_keygateway,
    "update-keygateway": update_keygateway,
    "supervisor-ssh": setup_supervisor_ssh,
    "dns-bootstrap": setup_dns_bootstrap,
    "stop-dns-bootstrap": teardown_dns_bootstrap,
    "bootstrap-registry": setup_bootstrap_registry,
    "update-registry": update_registry,
    "prometheus": setup_prometheus,
})
