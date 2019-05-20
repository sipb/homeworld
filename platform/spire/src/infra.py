import access
import command
import configuration
import setup
import ssh

import os
import traceback

def admit(server_principal: str) -> str:
    config = configuration.get_config()
    principal_hostname = config.get_fqdn(server_principal)

    errs = []

    try:
        if config.is_kerberos_enabled():
            return access.call_keyreq("bootstrap-token", principal_hostname).decode().strip()
    except Exception as e:
        print('[keyreq failed, set SPIRE_DEBUG for traceback]')
        if os.environ.get('SPIRE_DEBUG'):
            traceback.print_exc()
        errs.append(e)

    try:
        return ssh.check_ssh_output(config.keyserver, "keyinitadmit", principal_hostname).decode().strip()
    except Exception as e:
        print('[keyinitadmit failed, set SPIRE_DEBUG for traceback]')
        if os.environ.get('SPIRE_DEBUG'):
            traceback.print_exc()
        errs.append(e)

    if len(errs) > 1:
        raise command.MultipleExceptions('admit failed', errs)
    raise Exception('admit failed') from errs[0]


@command.wrap
def infra_admit(server_principal: str) -> None:
    "request a token to admit a node to the cluster"
    token = admit(server_principal)
    print("Token granted for %s: '%s'" % (server_principal, token))


@command.wrap
def infra_admit_all() -> None:
    "request tokens to admit every non-supervisor node to the cluster"
    config = configuration.get_config()
    tokens = []
    for node in config.nodes:
        if node.kind == "supervisor":
            continue
        token = admit(node.hostname)
        tokens.append((node.hostname, node.kind, str(node.ip), token))
    print('{:=^16} {:=^8} {:=^14} {:=^23}'.format('host', 'kind', 'ip', 'token'))
    for hostname, kind, ip, token in tokens:
        print('{:>16} {:^8} {:^14} {:<23}'.format(hostname, kind, ip, token))
    print('{:=^16} {:=^8} {:=^14} {:=^23}'.format('host', 'kind', 'ip', 'token'))


@command.wrapop
def infra_install_packages(ops: command.Operations) -> None:
    "install and update packages on a node"
    config = configuration.get_config()
    for node in config.nodes:
        setup.ssh_cmd(ops, "update apt repositories on @HOST", node, "apt-get", "update")
        setup.ssh_cmd(ops, "upgrade packages on @HOST", node, "apt-get", "dist-upgrade", "-y")


@command.wrapop
def infra_sync(ops: command.Operations, node_name: str) -> None:
    "synchronize the filesystem to disk on a node"
    node = configuration.get_config().get_node(node_name)
    setup.ssh_cmd(ops, "synchronize operations on @HOST", node, "sync")


@command.wrapop
def infra_sync_supervisor(ops: command.Operations) -> None:
    "synchronize the filesystem to disk on the supervisor"
    infra_sync(ops, configuration.get_config().keyserver.hostname)


main_command = command.Mux("commands about maintaining the infrastructure of a cluster", {
    "admit": infra_admit,
    "admit-all": infra_admit_all,
    "install-packages": infra_install_packages,
    "sync": infra_sync,
    "sync-supervisor": infra_sync_supervisor,
})
