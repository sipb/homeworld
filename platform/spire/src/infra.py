import access
import configuration
import command
import setup
import ssh


def admit(server_principal: str) -> str:
    config = configuration.get_config()
    principal_hostname = config.get_fqdn(server_principal)
    if config.is_kerberos_enabled():
        return access.call_keyreq("bootstrap-token", principal_hostname).decode().strip()
    else:
        return ssh.check_ssh_output(config.keyserver, "keyinitadmit", principal_hostname).decode().strip()


def infra_admit(server_principal: str) -> None:
    token = admit(server_principal)
    print("Token granted for %s: '%s'" % (server_principal, token))


def infra_admit_all() -> None:
    config = configuration.get_config()
    tokens = []
    for node in config.nodes:
        if node.kind == "supervisor":
            continue
        token = admit(node.hostname)
        tokens.append((node.hostname, node.kind, node.ip, token))
    print('{:=^16} {:=^8} {:=^14} {:=^23}'.format('host', 'kind', 'ip', 'token'))
    for hostname, kind, ip, token in tokens:
        print('{:>16} {:^8} {:^14} {:<23}'.format(hostname, kind, str(ip), token))
    print('{:=^16} {:=^8} {:=^14} {:=^23}'.format('host', 'kind', 'ip', 'token'))


def infra_install_packages(ops: setup.Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        ops.ssh("update apt repositories on @HOST", node, "apt-get", "update")
        ops.ssh("upgrade packages on @HOST", node, "apt-get", "dist-upgrade", "-y")


def infra_sync(ops: setup.Operations, node_name: str) -> None:
    node = configuration.get_config().get_node(node_name)
    ops.ssh("synchronize operations on @HOST", node, "sync")


def infra_sync_supervisor(ops: setup.Operations) -> None:
    infra_sync(ops, configuration.get_config().keyserver.hostname)


main_command = command.mux_map("commands about maintaining the infrastructure of a cluster", {
    "admit": command.wrap("request a token to admit a node to the cluster", infra_admit),
    "admit-all": command.wrap("request tokens to admit every non-supervisor node to the cluster", infra_admit_all),
    "install-packages": setup.wrapop("install and update packages on a node", infra_install_packages),
    "sync": setup.wrapop("synchronize the filesystem to disk on a node", infra_sync),
    "sync-supervisor": setup.wrapop("synchronize the filesystem to disk on the supervisor", infra_sync_supervisor),
})
