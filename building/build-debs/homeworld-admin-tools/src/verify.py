import query
import setup
import subprocess
import command
import configuration


def compare_multiline(a, b):
    a = a.split("\n")
    b = b.split("\n")
    for la, lb in zip(a, b):
        if la != lb:
            print("line mismatch (first):")
            print("<", la)
            print(">", lb)
            return False
    if len(a) != len(b):
        print("mismatched lengths of files")
        return False
    return True


def check_keystatics():
    machine_list = query.get_keyurl_data("/static/machine.list")
    expected_machine_list = configuration.get_machine_list_file()

    if not compare_multiline(machine_list, expected_machine_list):
        command.fail("MISMATCH: machine.list")

    cluster_conf = query.get_keyurl_data("/static/cluster.conf")
    expected_cluster_conf = configuration.get_cluster_conf()

    if not compare_multiline(cluster_conf, expected_cluster_conf):
        command.fail("MISMATCH: cluster.conf")

    print("pass: keyserver serving correct static files")


def check_online(server=None):
    config = configuration.Config.load_from_project()
    if server is None:
        found = config.nodes
        if not found:
            command.fail("no nodes configured")
    else:
        found = [node for node in config.nodes if
                 node.hostname == server or node.ip == server or node.hostname + "." + config.external_domain == server]
        if not found:
            command.fail("could not find server '%s' in setup.yaml" % server)
    any_offline = False
    for node in found:
        try:
            result = subprocess.check_output(["ssh", "root@%s.%s" % (node.hostname, config.external_domain),
                                          "echo round-trip"]).decode()
            is_online = (result == "round-trip\n")
        except subprocess.CalledProcessError:
            is_online = False
        if not is_online:
            any_offline = True
        print("NODE:", node.hostname.ljust(30), ("[ONLINE]" if is_online else "[OFFLINE]").rjust(10))
    if any_offline:
        command.fail("not all nodes were online!")
    print("All nodes: [ONLINE]")


main_command = command.mux_map("commands about verifying the state of a cluster", {
    "keystatics": command.wrap("verify that keyserver static files are being served properly", check_keystatics),
    "online": command.wrap("check whether a server (or all servers) is/are accepting SSH connections", check_online)
})
