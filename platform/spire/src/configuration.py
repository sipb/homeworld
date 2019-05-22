import os

import util
import command
import resource
import subprocess
import template
import version
import yaml


def get_project(create_dir_if_missing=False) -> str:
    project_dir = os.getenv("HOMEWORLD_DIR")
    if project_dir is None:
        command.fail("no HOMEWORLD_DIR environment variable declared")
    if not os.path.isdir(project_dir):
        if create_dir_if_missing:
            os.mkdir(project_dir)
        else:
            command.fail("HOMEWORLD_DIR (%s) is not a directory that exists" % project_dir)
    return project_dir


def get_editor() -> str:
    return os.getenv("EDITOR", "nano")


class CIDR:
    def __init__(self, cidr: str):
        if cidr.count("/") != 1:
            raise Exception("invalid cidr: %s" % cidr)
        ip, bits = cidr.split("/")
        bits = int(bits)
        if bits < 0 or bits > 32:
            raise Exception("invalid bits: %s" % bits)
        self.ip = IP(ip)
        self.bits = bits
        self.cidr = "%s/%d" % (self.ip.ip, self.bits)
        if (self.ip.to_integer() & self.addrmask_int()) != 0:
            raise Exception("CIDR includes lower bits that are set: %s" % cidr)

    def addrmask_int(self) -> int:
        return (1 << (32 - self.bits)) - 1

    def netmask_int(self) -> int:
        return self.addrmask_int() ^ 0xFFFFFFFF

    def netmask(self) -> "IP":
        return IP.from_integer(self.netmask_int())

    def gateway(self) -> "IP":
        if self.bits == 32:
            command.fail("no gateway can be assumed from a /32")
        return IP.from_integer(self.ip.to_integer() + 1)

    def __contains__(self, ip):
        return (ip & self.netmask()) == self.ip

    def __str__(self):
        return self.cidr

    def __repr__(self):
        return "cidr:" + self.cidr


class IP:
    def __init__(self, ip: str):
        if ip.count(".") != 3:
            raise Exception("invalid ipv4 address: %s" % ip)
        self.octets = [int(x) for x in ip.split(".")]
        for octet in self.octets:
            if octet < 0 or octet >= 256:
                raise Exception("invalid ipv4 address: bad octet %s" % octet)
        self.ip = "%d.%d.%d.%d" % tuple(self.octets)

    @classmethod
    def from_integer(cls, num: int) -> "IP":
        assert 0 <= num <= 0xFFFFFFFF
        octets = (((num & 0xFF000000) >> 24),
                  ((num & 0x00FF0000) >> 16),
                  ((num & 0x0000FF00) >> 8),
                  ((num & 0x000000FF) >> 0))
        return IP("%d.%d.%d.%d" % octets)

    def to_integer(self) -> int:
        return (self.octets[0] << 24) | (self.octets[1] << 16) | (self.octets[2] << 8) | self.octets[3]

    def __hash__(self):
        return hash(self.ip)

    def __eq__(self, other):
        if isinstance(other, IP):
            return self.ip == other.ip
        return NotImplemented

    def __ne__(self, other):
        if isinstance(other, IP):
            return self.ip != other.ip
        return NotImplemented

    def __and__(self, other):
        if isinstance(other, IP):
            return IP("%d.%d.%d.%d" % tuple(o1 & o2 for o1, o2 in zip(self.octets, other.octets)))
        else:
            return NotImplemented

    def __str__(self):
        return self.ip

    def __repr__(self):
        return "ip:" + self.ip


class Node:
    VALID_NODE_KINDS = {"master", "worker", "supervisor"}

    def __init__(self, config: dict):
        self.hostname, ip, self.kind = keycheck(config, "hostname", "ip", "kind")
        self.ip = IP(ip)

        if self.kind not in Node.VALID_NODE_KINDS:
            raise Exception("invalid node kind: %s" % self.kind)

    def __repr__(self):
        return "%s node %s (%s)" % (self.kind, self.hostname, self.ip)


def keycheck(kvs: dict, *keys: str, validator=lambda k, v: True):
    for key in kvs.keys():
        if key not in keys:
            command.fail("unexpected key %s in config" % key)
    for key in keys:
        if key not in kvs:
            command.fail("could not find expected key %s in config" % key)
    for key, value in kvs.items():
        if not validator(key, value):
            command.fail("config failed validation: key %s had invalid value %s" % (key, value))
    return [kvs[k] for k in keys]


class Config:
    def __init__(self, kv: dict):
        v_cluster, v_addresses, v_dns_upstreams, v_dns_bootstrap, v_root_admins, v_nodes = \
            keycheck(kv, "cluster", "addresses", "dns-upstreams", "dns-bootstrap", "root-admins", "nodes")

        self.external_domain, self.internal_domain, self.etcd_token, self.realm, self.mirror, self.user_grant_domain = \
            keycheck(v_cluster, "external-domain", "internal-domain", "etcd-token", "kerberos-realm", "mirror",
                                "user-grant-domain",
                     validator=lambda _, x: type(x) == str)

        cidr_nodes, cidr_pods, cidr_services, service_api, service_dns = \
            keycheck(v_addresses, "cidr-nodes", "cidr-pods", "cidr-services", "service-api", "service-dns",
                     validator=lambda _, x: type(x) == str)
        self.cidr_nodes = CIDR(cidr_nodes)
        self.cidr_pods = CIDR(cidr_pods)
        self.cidr_services = CIDR(cidr_services)
        self.service_api = IP(service_api)
        self.service_dns = IP(service_dns)

        if type(v_dns_upstreams) != list:
            command.fail("in config: expected dns-upstreams to be a list of IPs")
        self.dns_upstreams = [IP(server) for server in v_dns_upstreams]

        if self.service_api not in self.cidr_services or self.service_dns not in self.cidr_services:
            command.fail("in config: expected service IPs to be in the correct CIDR")

        self.dns_bootstrap = {hostname: IP(ip) for hostname, ip in v_dns_bootstrap.items()}

        self.root_admins = v_root_admins
        assert all(type(v) == str for v in self.root_admins)  # TODO: better error handling

        self.nodes = [Node(n) for n in v_nodes]

        self.keyserver = None

        for node in self.nodes:
            if node.kind == "supervisor":
                if self.keyserver is not None:
                    command.fail("in config: multiple supervisors not yet supported")
                self.keyserver = node

    def is_kerberos_enabled(self):
        return len(self.root_admins) > 0

    def has_node(self, node_name: str) -> bool:
        return any(node.hostname == node_name for node in self.nodes)

    def get_node(self, node_name: str) -> Node:
        for node in self.nodes:
            if node.hostname == node_name:
                return node
        command.fail("no such node: %s" % node_name)

    def get_fqdn(self, name: str) -> str:
        hostname = name
        if name.endswith("." + self.external_domain):
            # strip external domain
            hostname = name[:-(len(self.external_domain) + 1)]
        elif name.endswith("." + self.internal_domain):
            # strip internal domain
            hostname = name[:-(len(self.internal_domain) + 1)]
        if not self.has_node(hostname):
            command.fail("no such node: %s" % name)
        return hostname + "." + self.external_domain

    @classmethod
    def load_from_string(cls, contents: bytes) -> "Config":
        return Config(yaml.safe_load(contents))

    @classmethod
    def load_from_file(cls, filepath: str) -> "Config":
        return Config.load_from_string(util.readfile(filepath))

    @classmethod
    def get_setup_path(cls) -> str:
        return os.path.join(get_project(), "setup.yaml")

    @classmethod
    def load_from_project(cls) -> "Config":
        return Config.load_from_file(Config.get_setup_path())


def get_config() -> Config:
    return Config.load_from_project()


def get_keyserver_domain() -> str:
    config = Config.load_from_project()
    return config.keyserver.hostname + "." + config.external_domain


def get_etcd_endpoints() -> str:
    nodes = Config.load_from_project().nodes
    return ",".join("https://%s:2379" % n.ip for n in nodes if n.kind == "master")


def get_apiserver_default_as_node() -> Node:
    # TODO: this should be eliminated, because nothing should be specific to this one apiserver
    config = Config.load_from_project()
    apiservers = [node for node in config.nodes if node.kind == "master"]
    if not apiservers:
        command.fail("no apiserver to select, because no master nodes were configured")
    return apiservers[0]


def get_apiserver_default() -> str:
    return "https://%s:443" % get_apiserver_default_as_node().ip


def get_cluster_conf() -> str:
    config = Config.load_from_project()

    apiservers = [node for node in config.nodes if node.kind == "master"]

    cconf = {"APISERVER": get_apiserver_default(),
             "APISERVER_COUNT": len(apiservers),
             "CLUSTER_CIDR": config.cidr_pods,
             "CLUSTER_DOMAIN": config.internal_domain,
             "DOMAIN": config.external_domain,
             "ETCD_CLUSTER": ",".join("%s=https://%s:2380" % (n.hostname, n.ip) for n in apiservers),
             "ETCD_ENDPOINTS": get_etcd_endpoints(),
             "ETCD_TOKEN": config.etcd_token,
             "SERVICE_API": config.service_api,
             "SERVICE_CIDR": config.cidr_services,
             "SERVICE_DNS": config.service_dns}

    output = ["# generated by spire from setup.yaml\n"]
    output += ["%s=%s\n" % kv for kv in sorted(cconf.items())]
    return "".join(output)


def get_kube_cert_paths() -> (str, str, str):
    project_dir = get_project()
    return os.path.join(project_dir, "kube-access.key"),\
           os.path.join(project_dir, "kube-access.pem"),\
           os.path.join(project_dir, "kube-ca.pem")


def get_local_kubeconfig() -> str:
    key_path, cert_path, ca_path = get_kube_cert_paths()
    kconf = {"APISERVER": get_apiserver_default(),
             "AUTHORITY-PATH": ca_path,
             "CERT-PATH": cert_path,
             "KEY-PATH": key_path}
    return template.template("kubeconfig-local.yaml", kconf)


def get_prometheus_yaml() -> str:
    config = Config.load_from_project()
    kcli = {"APISERVER": get_apiserver_default_as_node().ip,
            "NODE-TARGETS": "[%s]" % ",".join("'%s.%s:9100'" % (node.hostname, config.external_domain)
                                              for node in config.nodes),
            "PULL-TARGETS": "[%s]" % ",".join("'%s.%s:9103'" % (node.hostname, config.external_domain)
                                              for node in config.nodes if node.kind != "supervisor"),
            "ETCD-TARGETS": "[%s]" % ",".join("'%s.%s:9101'" % (node.hostname, config.external_domain)
                                              for node in config.nodes if node.kind == "master")}
    return template.template("prometheus.yaml", kcli)


def populate() -> None:
    setup_yaml = os.path.join(get_project(create_dir_if_missing=True), "setup.yaml")
    if os.path.exists(setup_yaml):
        command.fail("setup.yaml already exists")
    resource.copy_to("setup.yaml", setup_yaml)
    print("filled out setup.yaml")


def edit() -> None:
    setup_yaml = os.path.join(get_project(), "setup.yaml")
    if not os.path.exists(setup_yaml):
        command.fail("setup.yaml does not exist (run spire config populate first?)")
    subprocess.check_call([get_editor(), "--", setup_yaml])


def print_cluster_conf() -> None:
    print(get_cluster_conf())


def print_local_kubeconfig() -> None:
    print(get_local_kubeconfig())


def print_prometheus_yaml() -> None:
    print(get_prometheus_yaml())


def get_kube_spec_vars(extra_kvs: dict=None) -> dict:
    config = Config.load_from_project()

    kvs = {
        "INTERNAL_DOMAIN": config.internal_domain,
        "NETWORK": config.cidr_pods,
        "SERVIP_API": config.service_api,
        "SERVIP_DNS": config.service_dns,
        # TODO: stop allowing use of just a single apiserver
        "SOME_APISERVER": [node for node in config.nodes if node.kind == "master"][0].ip,
    }
    if extra_kvs:
        kvs.update(extra_kvs)
    return kvs


def gen_kube_spec(output_name: str) -> None:
    util.writefile(output_name, get_single_kube_spec(os.path.basename(output_name)).encode())


def get_single_kube_spec(name: str, extra_kvs: dict=None) -> str:
    return template.template("clustered/%s" % name, get_kube_spec_vars(extra_kvs))


main_command = command.mux_map("commands about cluster configuration", {
    "populate": command.wrap("initialize the cluster's setup.yaml with the template", populate),
    "edit": command.wrap("open $EDITOR (defaults to nano) to edit the project's setup.yaml", edit),
    "gen-kube": command.wrap("generate kubernetes spec for the base cluster", gen_kube_spec),
    "show": command.mux_map("commands about showing different aspects of the configuration", {
        "cluster.conf": command.wrap("display the generated cluster.conf", print_cluster_conf),
        "kubeconfig": command.wrap("display the generated local kubeconfig", print_local_kubeconfig),
        "prometheus.yaml": command.wrap("display the generated prometheus.yaml", print_prometheus_yaml),
    }),
})
