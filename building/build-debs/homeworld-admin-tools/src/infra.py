import access
import configuration
import command
import setup


def infra_admit(server_principal: str) -> None:
    token = access.call_keyreq("bootstrap-token", server_principal, collect=True)
    print("Token granted for %s: '%s'" % (server_principal, token.decode().strip()))


def infra_admit_all() -> None:
    config = configuration.Config.load_from_project()
    tokens = {}
    for node in config.nodes:
        if node.kind == "supervisor":
            continue
        principal = node.hostname + "." + config.external_domain
        token = access.call_keyreq("bootstrap-token", principal, collect=True).decode().strip()
        tokens[node.hostname] = (node.kind, node.ip, token)
    print("host".center(16, "="), "kind".center(8, "="), "ip".center(14, "="), "token".center(21, "="))
    for key, (kind, ip, token) in sorted(tokens.items()):
        print(key.rjust(16), kind.center(8), str(ip).center(14), token.ljust(21))
    print("host".center(16, "="), "kind".center(8, "="), "ip".center(14, "="), "token".center(21, "="))


def infra_install_packages(ops: setup.Operations, config: configuration.Config) -> None:
    for node in config.nodes:
        ops.ssh("update apt repositories on @HOST", node, "apt-get", "update")
        ops.ssh("upgrade packages on @HOST", node, "apt-get", "upgrade", "-y")
        if node.kind == "supervisor":
            ops.ssh("install supervisor packages on @HOST", node, "apt-get", "install", "-y",
                    "homeworld-bootstrap-registry")
        else:
            ops.ssh("install standard packages on @HOST", node, "apt-get", "install", "-y",
                    "homeworld-services")


main_command = command.mux_map("commands about maintaining the infrastructure of a cluster", {
    "admit": command.wrap("request a token to admit a node to the cluster", infra_admit),
    "admit-all": command.wrap("request tokens to admit every non-supervisor node to the cluster", infra_admit_all),
    "install-packages": setup.wrapop("install and update packages on a node", infra_install_packages),
})
