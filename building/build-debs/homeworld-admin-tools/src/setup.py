import os
import subprocess
import tempfile
import time

import authority
import command
import configuration
import resource
import util


def escape_shell(param: str) -> str:
    # replaces ' -> '"'"'
    return "'" + param.replace("'", "'\"'\"'") + "'"


class Operations:
    def __init__(self, config: configuration.Config):
        self._ops = []
        self._config = config

    def add_operation(self, name: str, callback, node: configuration.Node=None) -> None:
        if node:
            name = name.replace("@HOST", node.hostname)
        self._ops.append((name, callback))

    def run_operations(self) -> None:
        print("== executing %d operations ==" % len(self._ops))
        print()
        startat = time.time()
        for i, (name, operation) in enumerate(self._ops, 1):
            print("--", name, " (%d/%d) --" % (i, len(self._ops)))
            operation()
            print()
        print("== all operations executed in %.2f seconds! ==" % (time.time() - startat))

    def subprocess(self, name: str, *argv: str, node: configuration.Node=None) -> None:
        self.add_operation(name, lambda: subprocess.check_call(argv), node=node)

    def pause(self, name: str, duration):
        self.add_operation(name, lambda: time.sleep(duration))

    def _ssh_get_login(self, node: configuration.Node) -> str:  # returns root@<HOSTNAME>.<EXTERNAL_DOMAIN>
        return "root@%s.%s" % (node.hostname, self._config.external_domain)

    def ssh_raw(self, name: str, node: configuration.Node, script: str, in_directory: str=None, redirect_to: str=None)\
            -> None:
        if redirect_to:
            script = "(%s) >%s" % (script, escape_shell(redirect_to))
        if in_directory:
            script = "cd %s && %s" % (escape_shell(in_directory), script)
        self.subprocess(name, "ssh", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=1", self._ssh_get_login(node), script, node=node)

    def ssh(self, name: str, node: configuration.Node, *argv: str, in_directory: str=None, redirect_to: str=None)\
            -> None:
        self.ssh_raw(name, node, " ".join(escape_shell(param) for param in argv),
                     in_directory=in_directory, redirect_to=redirect_to)

    def ssh_mkdir(self, name: str, node: configuration.Node, *paths: str, with_parents: bool=True) -> None:
        options = ["-p"] if with_parents else []
        self.ssh(name, node, "mkdir", *(options + ["--"] + list(paths)))

    def ssh_upload_path(self, name: str, node: configuration.Node, source_path: str, dest_path: str) -> None:
        self.subprocess(name, "scp", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=1", "--", source_path, self._ssh_get_login(node) + ":" + dest_path, node=node)

    def ssh_upload_bytes(self, name: str, node: configuration.Node, source_bytes: bytes, dest_path: str) -> None:
        dest_rel = self._ssh_get_login(node) + ":" + dest_path

        def upload_bytes() -> None:
            # tempfile.TemporaryDirectory() creates the directory with 0o600, which protects the data if it's sensitive
            with tempfile.TemporaryDirectory() as scratchdir:
                scratchpath = os.path.join(scratchdir, "scratch")
                util.writefile(scratchpath, source_bytes)
                subprocess.check_call(["scp", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=1", "--", scratchpath, dest_rel])
                os.remove(scratchpath)

        self.add_operation(name, upload_bytes, node=node)


AUTHORITY_DIR = "/etc/homeworld/keyserver/authorities"
STATICS_DIR = "/etc/homeworld/keyserver/static"
CONFIG_DIR = "/etc/homeworld/config"
KEYCLIENT_DIR = "/etc/homeworld/keyclient"
KEYTAB_PATH = "/etc/krb5.keytab"


def setup_keyserver(ops: Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ops.ssh_mkdir("create directories on @HOST", node, AUTHORITY_DIR, STATICS_DIR, CONFIG_DIR)
        ops.ssh_upload_path("upload authorities to @HOST", node,
                            authority.get_targz_path(), AUTHORITY_DIR + "/authorities.tgz")
        ops.ssh_raw("extract authorities on @HOST", node,
                    "tar -xzf authorities.tgz && rm authorities.tgz", in_directory=AUTHORITY_DIR)
        ops.ssh_upload_bytes("upload cluster config to @HOST", node,
                             configuration.get_cluster_conf().encode(), STATICS_DIR + "/cluster.conf")
        ops.ssh_upload_bytes("upload machine list to @HOST", node,
                             configuration.get_machine_list_file().encode(), STATICS_DIR + "/machine.list")
        ops.ssh_upload_bytes("upload keyserver config to @HOST", node,
                             configuration.get_keyserver_yaml().encode(), CONFIG_DIR + "/keyserver.yaml")
        ops.ssh("start keyserver on @HOST", node, "systemctl", "restart", "keyserver.service")


def admit_keyserver(ops: Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        domain = node.hostname + "." + config.external_domain
        ops.ssh("request bootstrap token for @HOST", node,
                "keyinitadmit", CONFIG_DIR + "/keyserver.yaml", domain, "bootstrap-keyinit",
                redirect_to=KEYCLIENT_DIR + "/bootstrap.token")
        # TODO: do we need to poke the keyclient to make sure it tries again?
        # TODO: don't wait four seconds if it isn't necessary
        ops.ssh("kick keyclient daemon on @HOST", node, "systemctl", "restart", "keyclient")
        ops.pause("giving admission time to complete...", 4.0)  # 4 seconds
        # if it doesn't exist, this command will fail.
        ops.ssh("confirm that @HOST was admitted", node, "test", "-e", KEYCLIENT_DIR + "/granting.pem")


def setup_keygateway(ops: Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        keytab = os.path.join(configuration.get_project(), "keytab.%s" % node.hostname)
        ops.ssh("confirm no existing keytab on @HOST", node, "test", "!", "-e", KEYTAB_PATH)
        ops.ssh_upload_path("upload keytab for @HOST", node, keytab, KEYTAB_PATH)
        ops.ssh("restart keygateway on @HOST", node, "systemctl", "restart", "keygateway")


def setup_supervisor_ssh(ops: Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_config = resource.get_resource("sshd_config")
        ops.ssh_upload_bytes("upload new ssh configuration to @HOST", node, ssh_config, "/etc/ssh/sshd_config")
        ops.ssh("reload ssh configuration on @HOST", node, "systemctl", "restart", "ssh")
        ops.ssh("shift aside old authorized_keys on @HOST", node,
                "mv", "/root/.ssh/authorized_keys", "/root/original_authorized_keys")


def setup_services(ops: Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        if node.kind == "master":
            ops.ssh("start etcd on master @HOST", node, "/usr/lib/hyades/start-master-etcd.sh")
    ops.pause("pause for etcd startup", 2)
    for node in config.nodes:
        if node.kind == "master":
            ops.ssh("start all on master @HOST", node, "/usr/lib/hyades/start-master.sh")
    ops.pause("pause for kubernetes startup", 2)
    for node in config.nodes:
        if node.kind == "worker":
            ops.ssh("start all on worker @HOST", node, "/usr/lib/hyades/start-worker.sh")


def modify_dns_bootstrap(ops: Operations, config: configuration.Config, is_install: bool) -> None:
    for node in config.nodes:
        if node.kind == "supervisor":
            continue
        strip_cmd = "grep -vF AUTO-HOMEWORLD-BOOTSTRAP /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts"
        ops.ssh_raw("strip bootstrapped dns on @HOST", node, strip_cmd)
        if is_install:
            for hostname, ip in config.dns_bootstrap.items():
                new_hosts_line = "%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP" % (ip, hostname)
                strip_cmd = "echo %s >>/etc/hosts" % escape_shell(new_hosts_line)
                ops.ssh_raw("bootstrap dns on @HOST: %s" % hostname, node, strip_cmd)


def setup_dns_bootstrap(ops: Operations, config: configuration.Config) -> None:
    modify_dns_bootstrap(ops, config, True)


def teardown_dns_bootstrap(ops: Operations, config: configuration.Config) -> None:
    modify_dns_bootstrap(ops, config, False)


REGISTRY_HOSTNAME = "homeworld.mit.edu"


def setup_bootstrap_registry(ops: Operations, config: configuration.Config) -> None:
    https_cert_dir = os.path.join(configuration.get_project(), "https-certs")
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        keypath = os.path.join(https_cert_dir, "%s.key" % REGISTRY_HOSTNAME)
        certpath = os.path.join(https_cert_dir, "%s.pem" % REGISTRY_HOSTNAME)

        ops.ssh_mkdir("create ssl cert directory on @HOST", node, "/etc/homeworld/ssl")
        ops.ssh_upload_path("upload %s key to @HOST" % REGISTRY_HOSTNAME, node, keypath, "/etc/homeworld/ssl/%s.key" % REGISTRY_HOSTNAME)
        ops.ssh_upload_path("upload %s cert to @HOST" % REGISTRY_HOSTNAME, node, certpath, "/etc/homeworld/ssl/%s.pem" % REGISTRY_HOSTNAME)
        ops.ssh("restart nginx on @HOST", node, "systemctl", "restart", "nginx")


def wrapop(desc: str, f):
    def wrap_param_tx(params):
        config = configuration.Config.load_from_project()
        ops = Operations(config)
        return [ops, config] + params, ops.run_operations
    return command.wrap(desc, f, wrap_param_tx)


main_command = command.mux_map("commands about setting up a cluster", {
    "keyserver": wrapop("deploy keys and configuration for keyserver; start keyserver", setup_keyserver),
    "self-admit": wrapop("admit the keyserver into the cluster during bootstrapping", admit_keyserver),
    "keygateway": wrapop("deploy keytab and start keygateway", setup_keygateway),
    "supervisor-ssh": wrapop("configure supervisor SSH access", setup_supervisor_ssh),
    "services": wrapop("bring up all cluster services in sequence", setup_services),
    "dns-bootstrap": wrapop("switch cluster nodes into 'bootstrapped DNS' mode", setup_dns_bootstrap),
    "stop-dns-bootstrap": wrapop("switch cluster nodes out of 'bootstrapped DNS' mode", teardown_dns_bootstrap),
    "bootstrap-registry": wrapop("bring up the bootstrap container registry on the supervisor nodes", setup_bootstrap_registry),
})
