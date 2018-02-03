import access
import configuration
import command
import setup


def infra_admit(server_principal: str) -> None:
    config = configuration.get_config()
    principal_hostname = config.get_fqdn(server_principal)
    token = access.call_keyreq("bootstrap-token", principal_hostname, collect=True)
    print("Token granted for %s: '%s'" % (server_principal, token.decode().strip()))


def infra_admit_all() -> None:
    config = configuration.get_config()
    tokens = {}
    for node in config.nodes:
        if node.kind == "supervisor":
            continue
        principal = node.hostname + "." + config.external_domain
        token = access.call_keyreq("bootstrap-token", principal, collect=True).decode().strip()
        tokens[node.hostname] = (node.kind, node.ip, token)
    print("host".center(16, "="), "kind".center(8, "="), "ip".center(14, "="), "token".center(23, "="))
    for key, (kind, ip, token) in sorted(tokens.items()):
        print(key.rjust(16), kind.center(8), str(ip).center(14), token.ljust(23))
    print("host".center(16, "="), "kind".center(8, "="), "ip".center(14, "="), "token".center(23, "="))


def infra_install_packages(ops: setup.Operations) -> None:
    config = configuration.get_config()
    for node in config.nodes:
        ops.ssh("update apt repositories on @HOST", node, "apt-get", "update")
        ops.ssh("upgrade packages on @HOST", node, "apt-get", "upgrade", "-y")


main_command = command.mux_map("commands about maintaining the infrastructure of a cluster", {
    "admit": command.wrap("request a token to admit a node to the cluster", infra_admit),
    "admit-all": command.wrap("request tokens to admit every non-supervisor node to the cluster", infra_admit_all),
    "install-packages": setup.wrapop("install and update packages on a node", infra_install_packages),
})
