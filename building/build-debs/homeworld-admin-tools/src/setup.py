import time
import tempfile

import os

import authority
import subprocess
import configuration
import command
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

    def _ssh_get_login(self, node: configuration.Node) -> str:  # returns root@<HOSTNAME>.<EXTERNAL_DOMAIN>
        return "root@%s.%s" % (node.hostname, self._config.external_domain)

    def ssh_raw(self, name: str, node: configuration.Node, script: str, in_directory: str=None) -> None:
        if in_directory:
            script = "cd %s && %s" % (escape_shell(in_directory), script)
        self.subprocess(name, "ssh", self._ssh_get_login(node), script, node=node)

    def ssh(self, name: str, node: configuration.Node, *argv: str, in_directory: str=None) -> None:
        self.ssh_raw(name, node, " ".join(escape_shell(param) for param in argv), in_directory=in_directory)

    def ssh_mkdir(self, name: str, node: configuration.Node, *paths: str, with_parents: bool=True) -> None:
        options = ["-p"] if with_parents else []
        self.ssh(name, node, "mkdir", *(options + ["--"] + list(paths)))

    def ssh_upload_path(self, name: str, node: configuration.Node, source_path: str, dest_path: str) -> None:
        self.subprocess(name, "scp", "--", source_path, self._ssh_get_login(node) + ":" + dest_path, node=node)

    def ssh_upload_bytes(self, name: str, node: configuration.Node, source_bytes: bytes, dest_path: str) -> None:
        dest_rel = self._ssh_get_login(node) + ":" + dest_path

        def upload_bytes() -> None:
            # tempfile.TemporaryDirectory() creates the directory with 0o600, which protects the data if it's sensitive
            with tempfile.TemporaryDirectory() as scratchdir:
                scratchpath = os.path.join(scratchdir, "scratch")
                util.writefile(scratchpath, source_bytes)
                subprocess.check_call(["scp", "--", scratchpath, dest_rel])
                os.remove(scratchpath)

        self.add_operation(name, upload_bytes, node=node)


AUTHORITY_DIR = "/etc/homeworld/keyserver/authorities"
STATICS_DIR = "/etc/homeworld/keyserver/static"
CONFIG_DIR = "/etc/homeworld/config"


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


def wrapop(desc: str, f):
    def wrap_param_tx(params):
        config = configuration.Config.load_from_project()
        ops = Operations(config)
        return [ops, config] + params, ops.run_operations
    return command.wrap(desc, f, wrap_param_tx)


main_command = command.mux_map("commands about setting up a cluster", {
    "keyserver": wrapop("deploy keys and configuration for keyserver; start keyserver", setup_keyserver),
})
