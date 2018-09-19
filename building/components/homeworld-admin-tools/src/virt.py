import os
import subprocess
import tempfile

import command
import configuration
import seq


def get_bridge(ip):
    return "spirebr%s" % hex(ip.to_integer())[2:].upper()


def get_node_tap(node):
    # maximum length: 15 characters
    return "spirtap%s" % hex(node.ip.to_integer())[2:].upper()


def determine_topology():
    config = configuration.get_config()
    gateway_ip = config.cidr_nodes.gateway()
    gateway = "%s/%d" % (gateway_ip, config.cidr_nodes.bits)
    taps = []
    hosts = {}
    for node in config.nodes:
        if node.ip not in config.cidr_nodes:
            command.fail("invalid topology: address %s is not in CIDR %s" % (node.ip, config.cidr_nodes))
        taps.append(get_node_tap(node))
        hosts["%s.%s" % (node.hostname, config.external_domain)] = node.ip
    return gateway, taps, get_bridge(gateway_ip), hosts


def sudo(*command):
    subprocess.check_call(["sudo"] + list(command))


def sudo_ok(*command):
    return subprocess.call(["sudo"] + list(command)) == 0


def sysctl_set(key, value):
    sudo("sysctl", "-w", "--", "%s=%s" % (key, value))


def bridge_up(bridge_name, address):
    sudo("brctl", "addbr", bridge_name)
    sudo("ip", "link", "set", bridge_name, "up")
    sudo("ip", "addr", "add", address, "dev", bridge_name)


def bridge_down(bridge_name, address):
    ok = sudo_ok("ip", "addr", "del", address, "dev", bridge_name)
    ok &= sudo_ok("ip", "link", "set", bridge_name, "down")
    ok &= sudo_ok("brctl", "delbr", bridge_name)
    return ok


def tap_up(bridge_name, tap):
    sudo("ip", "tuntap", "add", "user", os.getenv("USER"), "mode", "tap", tap)
    sudo("ip", "link", "set", tap, "up", "promisc", "on")
    sudo("brctl", "addif", bridge_name, tap)


def tap_down(bridge_name, tap):
    ok = sudo_ok("brctl", "delif", bridge_name, tap)
    ok &= sudo_ok("ip", "link", "set", tap, "down")
    ok &= sudo_ok("ip", "tuntap", "del", "mode", "tap", tap)
    return ok


def does_link_exist(link):
    return subprocess.check_call(["ip", "link", "show", "dev", link],
                                 stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL) == 0


def get_upstream_link():
    lines = subprocess.check_output(["ip", "-o", "-d", "route"]).decode().split("\n")
    defaults = [line for line in lines if line.startswith("unicast default via")]
    if len(defaults) != 1:
        command.fail("cannot determine upstream link from ip route output")
    link = defaults[0].split(" dev ")[1].split(" ")[0]
    if not does_link_exist(link):
        command.fail("failed to correctly determine upstream link: '%s' does not exist" % link)
    return link


def routing_up(bridge_name, upstream_link):
    sudo("iptables", "-I", "INPUT", "1", "-i", bridge_name, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", bridge_name, "-o", upstream_link, "-j", "ACCEPT")
    sudo("iptables", "-I", "FORWARD", "1", "-i", upstream_link, "-o", bridge_name, "-j", "ACCEPT")
    sudo("iptables", "-t", "nat", "-I", "POSTROUTING", "1", "-o", upstream_link, "-j", "MASQUERADE")


def routing_down(bridge_name, upstream_link):
    ok = sudo_ok("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", upstream_link, "-j", "MASQUERADE")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", upstream_link, "-o", bridge_name, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "FORWARD", "-i", bridge_name, "-o", upstream_link, "-j", "ACCEPT")
    ok &= sudo_ok("iptables", "-D", "INPUT", "-i", bridge_name, "-j", "ACCEPT")
    return ok


def sudo_update_file_by_filter(filename, discard_predicate):
    with tempfile.NamedTemporaryFile(mode="w") as fw:
        with open(filename, "r") as fr:
            for line in fr:
                line = line.rstrip("\n")
                if not discard_predicate(line):
                    fw.write(line + "\n")
        fw.flush()
        sudo("cp", "--", fw.name, filename)


def sudo_append_to_file(filename, lines):
    subprocess.run(["sudo", "tee", "-a", "--", filename], stdout=subprocess.DEVNULL, check=True,
                          input="".join(("%s\n" % line) for line in lines).encode())


def hosts_up(hosts):
    for host, ip in hosts.items():
        if "\t" in host:
            command.fail("expected no tabs in hostname %s" % repr(host))
        assert "\t" not in str(ip)
    sudo_append_to_file("/etc/hosts", ["%s\t%s" % (ip, hostname) for hostname, ip in hosts.items()])


def hosts_down(hosts):
    def is_our_host(line):
        if line.count("\t") != 1:
            return False
        ip, hostname = line.split("\t")
        return hostname in hosts and str(hosts[hostname]) == ip
    sudo_update_file_by_filter("/etc/hosts", discard_predicate=is_our_host)


def net_up_inner(gateway_ip, taps, bridge_name, hosts):
    upstream_link = get_upstream_link()

    sysctl_set("net.ipv4.ip_forward", 1)

    try:
        bridge_up(bridge_name, gateway_ip)
        for tap in taps:
            tap_up(bridge_name, tap)

        routing_up(bridge_name, upstream_link)

        hosts_up(hosts)
    except Exception as e:
        print("woops, tearing down...")
        if not net_down():
            print("could not tear down")
        raise e


def net_down_inner(gateway_ip, taps, bridge_name, hosts, fail=False):
    upstream_link = get_upstream_link()

    hosts_down(hosts)

    ok = routing_down(bridge_name, upstream_link)

    for tap in taps:
        ok &= tap_down(bridge_name, tap)

    ok &= bridge_down(bridge_name, gateway_ip)

    if not ok and fail:
        command.fail("tearing down network failed (maybe it was already torn down?)")
    return ok


def net_up():
    gateway_ip, taps, bridge_name, hosts = determine_topology()
    net_up_inner(gateway_ip, taps, bridge_name, hosts)


def net_down(fail=False):
    gateway_ip, taps, bridge_name, hosts = determine_topology()
    return net_down_inner(gateway_ip, taps, bridge_name, hosts, fail)

main_command = seq.seq_mux_map("commands to run local testing VMs", {
    "net": command.mux_map("commands to control the state of the local testing network", {
        "up": command.wrap("bring up local testing network", net_up),
        "down": command.wrap("bring down local testing network", net_down),
    }),
})
