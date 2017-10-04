import query
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


main_command = command.mux_map("commands about verifying the state of a cluster", {
    "keystatics": command.wrap("verify that keyserver static files are being served properly", check_keystatics),
})
