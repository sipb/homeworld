import contextlib
import os
import subprocess
import sys
import time

import authority
import command
import configuration
import keycrypt
import resource
import ssh


def escape_shell(param: str) -> str:
    # replaces ' -> '"'"'
    return "'" + param.replace("'", "'\"'\"'") + "'"


class Operations:
    def __init__(self):
        self._ops = []
        self._annotations = []
        self._cleanup = []
        self._ignore_annotations = 0

    def add_operation(self, name: str, callback, node: configuration.Node=None) -> None:
        if node:
            name = name.replace("@HOST", node.hostname)
        self.annotate_from(callback)
        self._ops.append((name, callback))

    def add_subcommand(self, func):
        self.annotate_from(func)
        self._ignore_annotations += 1
        func(self)
        self._ignore_annotations -= 1

    def context(self, name, ctx):
        # TODO: figure out annotations so that this can be processed correctly by --dry-run
        def enter():
            ctx.__enter__()
            self._cleanup.append(ctx.__exit__)

        def subctx():
            self.add_operation("enter context: %s" % name, enter)
            yield ctx
            self.add_operation("exit context: %s" % name, self.context_cleanup)
        return contextlib.contextmanager(subctx)()

    def annotate_from(self, func):
        self.annotate_subcommand(command.get_command_for_function(func))

    def annotate_subcommand(self, command):
        if not self._ignore_annotations:
            self._annotations.append("$ " + " ".join(command))

    def print_annotations(self):
        if not self._ignore_annotations:
            for annotation in self._annotations:
                print("  %s" % annotation)

    def context_cleanup(self, exc_info=(None, None, None)):
        if not self._cleanup:
            raise Exception("internal error: context manager stack underflow")
        if self._cleanup.pop()(*exc_info):
            print("unsupported: context manager attempted to suppress exception")

    def run_operations(self) -> None:
        print("== executing %d operations ==" % len(self._ops))
        print()
        if self._cleanup:
            raise Exception("attempt to reuse Operations object that was never fully cleaned up")
        startat = time.time()
        try:
            for i, (name, operation) in enumerate(self._ops, 1):
                print("--", name, " (%d/%d) --" % (i, len(self._ops)))
                operation()
                print()
            if self._cleanup:
                raise Exception("Operations object failed to fully clean up during operation sequence")
        finally:
            if self._cleanup:
                print("== running %d cleanup operations ==" % len(self._cleanup))
                exc_info = sys.exc_info()
                while self._cleanup:
                    self.context_cleanup(exc_info)

        print("== all operations executed in %.2f seconds! ==" % (time.time() - startat))

    def pause(self, name: str, duration):
        self.add_operation(name, lambda: time.sleep(duration))

    def ssh_raw(self, name: str, node: configuration.Node, script: str, in_directory: str=None, redirect_to: str=None)\
            -> None:
        if redirect_to:
            script = "(%s) >%s" % (script, escape_shell(redirect_to))
        if in_directory:
            script = "cd %s && %s" % (escape_shell(in_directory), script)
        self.add_operation(name, lambda: ssh.check_ssh(node, script), node=node)

    def ssh(self, name: str, node: configuration.Node, *argv: str, in_directory: str=None, redirect_to: str=None)\
            -> None:
        self.ssh_raw(name, node, " ".join(escape_shell(param) for param in argv),
                     in_directory=in_directory, redirect_to=redirect_to)

    def ssh_mkdir(self, name: str, node: configuration.Node, *paths: str, with_parents: bool=True) -> None:
        options = ["-p"] if with_parents else []
        self.ssh(name, node, "mkdir", *(options + ["--"] + list(paths)))

    def ssh_upload_path(self, name: str, node: configuration.Node, source_path: str, dest_path: str) -> None:
        self.add_operation(name, lambda: ssh.check_scp_up(node, source_path, dest_path), node=node)

    def ssh_upload_bytes(self, name: str, node: configuration.Node, source_bytes: bytes, dest_path: str) -> None:
        self.add_operation(name, lambda: ssh.upload_bytes(node, source_bytes, dest_path), node=node)


AUTHORITY_DIR = "/etc/homeworld/keyserver/authorities"
STATICS_DIR = "/etc/homeworld/keyserver/static"
CONFIG_DIR = "/etc/homeworld/config"
KEYCLIENT_DIR = "/etc/homeworld/keyclient"
KEYTAB_PATH = "/etc/krb5.keytab"


def setup_keyserver(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ops.ssh_mkdir("create directories on @HOST", node, AUTHORITY_DIR, STATICS_DIR, CONFIG_DIR)
        for name, data in authority.iterate_keys_decrypted():
            # TODO: keep these keys in memory
            if "/" in name:
                command.fail("found key in upload list with invalid filename")
            # TODO: avoid keeping these keys in memory for this long
            ops.ssh_upload_bytes("upload authority %s to @HOST" % name, node, data, os.path.join(AUTHORITY_DIR, name))
        ops.ssh_upload_bytes("upload cluster config to @HOST", node,
                             configuration.get_cluster_conf().encode(), STATICS_DIR + "/cluster.conf")
        ops.ssh_upload_bytes("upload machine list to @HOST", node,
                             configuration.get_machine_list_file().encode(), STATICS_DIR + "/machine.list")
        ops.ssh_upload_bytes("upload keyserver config to @HOST", node,
                             configuration.get_keyserver_yaml().encode(), CONFIG_DIR + "/keyserver.yaml")
        ops.ssh("enable keyserver on @HOST", node, "systemctl", "enable", "keyserver.service")
        ops.ssh("start keyserver on @HOST", node, "systemctl", "restart", "keyserver.service")


def admit_keyserver(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        domain = node.hostname + "." + config.external_domain
        ops.ssh("request bootstrap token for @HOST", node,
                "keyinitadmit", CONFIG_DIR + "/keyserver.yaml", domain, domain, "bootstrap-keyinit",
                redirect_to=KEYCLIENT_DIR + "/bootstrap.token")
        # TODO: do we need to poke the keyclient to make sure it tries again?
        # TODO: don't wait four seconds if it isn't necessary
        ops.ssh("kick keyclient daemon on @HOST", node, "systemctl", "restart", "keyclient")
        # if it doesn't exist, this command will fail.
        ops.ssh("confirm that @HOST was admitted", node, "test", "-e", KEYCLIENT_DIR + "/granting.pem")
        ops.ssh("enable auth-monitor daemon on @HOST", node, "systemctl", "enable", "auth-monitor")
        ops.ssh("start auth-monitor daemon on @HOST", node, "systemctl", "restart", "auth-monitor")


def modify_keygateway(ops: Operations, overwrite_keytab: bool) -> None:
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
        ops.add_operation("upload keytab for @HOST", safe_upload_keytab, node)
        ops.ssh("enable keygateway on @HOST", node, "systemctl", "enable", "keygateway")
        ops.ssh("restart keygateway on @HOST", node, "systemctl", "restart", "keygateway")


def setup_keygateway(ops: Operations) -> None:
    modify_keygateway(ops, False)


def update_keygateway(ops: Operations) -> None:
    modify_keygateway(ops, True)


def setup_supervisor_ssh(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ssh_config = resource.get_resource("sshd_config")
        ops.ssh_upload_bytes("upload new ssh configuration to @HOST", node, ssh_config, "/etc/ssh/sshd_config")
        ops.ssh("reload ssh configuration on @HOST", node, "systemctl", "restart", "ssh")
        ops.ssh_raw("shift aside old authorized_keys on @HOST", node,
                "if [ -f /root/.ssh/authorized_keys ]; then " +
                "mv /root/.ssh/authorized_keys " +
                "/root/original_authorized_keys; fi")


def modify_dns_bootstrap(ops: Operations, is_install: bool) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        strip_cmd = "grep -vF AUTO-HOMEWORLD-BOOTSTRAP /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts"
        ops.ssh_raw("strip bootstrapped dns on @HOST", node, strip_cmd)
        if is_install:
            for hostname, ip in config.dns_bootstrap.items():
                new_hosts_line = "%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP" % (ip, hostname)
                strip_cmd = "echo %s >>/etc/hosts" % escape_shell(new_hosts_line)
                ops.ssh_raw("bootstrap dns on @HOST: %s" % hostname, node, strip_cmd)


def dns_bootstrap_lines() -> str:
    config = configuration.get_config()
    return "".join("%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP\n" % (ip, hostname) for hostname, ip in config.dns_bootstrap.items())


def setup_dns_bootstrap(ops: Operations) -> None:
    modify_dns_bootstrap(ops, True)


def teardown_dns_bootstrap(ops: Operations) -> None:
    modify_dns_bootstrap(ops, False)


def setup_bootstrap_registry(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue

        ops.ssh("unmask nginx on @HOST", node, "systemctl", "unmask", "nginx")
        ops.ssh("enable nginx on @HOST", node, "systemctl", "enable", "nginx")
        ops.ssh("restart nginx on @HOST", node, "systemctl", "restart", "nginx")


def setup_prometheus(ops: Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind != "supervisor":
            continue
        ops.ssh_upload_bytes("upload prometheus config to @HOST", node, configuration.get_prometheus_yaml().encode(),
                             "/etc/prometheus.yaml")
        ops.ssh("enable prometheus on @HOST", node, "systemctl", "enable", "prometheus")
        ops.ssh("restart prometheus on @HOST", node, "systemctl", "restart", "prometheus")


def wrapop(desc: str, f):
    def wrap_param_tx(args):
        ops = Operations()
        return [ops] + args.params, ops.run_operations
    return command.wrap(desc, f, wrap_param_tx)


main_command = command.mux_map("commands about setting up a cluster", {
    "keyserver": wrapop("deploy keys and configuration for keyserver; start keyserver", setup_keyserver),
    "self-admit": wrapop("admit the keyserver into the cluster during bootstrapping", admit_keyserver),
    "keygateway": wrapop("deploy keytab and start keygateway", setup_keygateway),
    "update-keygateway": wrapop("update keytab and restart keygateway", update_keygateway),
    "supervisor-ssh": wrapop("configure supervisor SSH access", setup_supervisor_ssh),
    "dns-bootstrap": wrapop("switch cluster nodes into 'bootstrapped DNS' mode", setup_dns_bootstrap),
    "stop-dns-bootstrap": wrapop("switch cluster nodes out of 'bootstrapped DNS' mode", teardown_dns_bootstrap),
    "bootstrap-registry": wrapop("bring up the bootstrap container registry on the supervisor nodes", setup_bootstrap_registry),
    "prometheus": wrapop("bring up the supervisor node prometheus instance", setup_prometheus),
})
