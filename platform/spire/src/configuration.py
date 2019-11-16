from ipaddress import IPv4Address, IPv4Network
import jsonschema
import os
import subprocess
import yaml

import command
import resource
import resources
import template
import util


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


class Node:
    VALID_NODE_KINDS = {"master", "worker", "supervisor"}

    def __init__(self, config: dict):
        self.hostname = config["hostname"]
        self.kind = config["kind"]
        self.ip = IPv4Address(config["ip"])

        if self.kind not in Node.VALID_NODE_KINDS:
            raise Exception("invalid node kind: %s" % self.kind)

    def __repr__(self):
        return "%s node %s (%s)" % (self.kind, self.hostname, self.ip)


SCHEMA = yaml.safe_load(resources.get_resource("setup-schema.yaml"))


class Config:
    def __init__(self, kv: dict):
        jsonschema.validate(kv, SCHEMA)

        self.external_domain = kv["cluster"]["external-domain"]
        self.internal_domain = kv["cluster"]["internal-domain"]
        self.etcd_token = kv["cluster"]["etcd-token"]
        self.realm = kv["cluster"]["kerberos-realm"]
        self.mirror = kv["cluster"]["mirror"]
        self.user_grant_domain = kv["cluster"]["user-grant-domain"]
        self.user_grant_email_domain = kv["cluster"]["user-grant-email-domain"]

        # the vlan on the trunk that each server needs to attach to in order to access the internet. "0" to represent
        # that the server is either not on a vlan or that the VLAN is untrunked.
        self.vlan = kv.get("vlan", 0)

        self.cidr_nodes = IPv4Network(kv["addresses"]["cidr-nodes"])
        self.cidr_pods = IPv4Network(kv["addresses"]["cidr-pods"])
        self.cidr_services = IPv4Network(kv["addresses"]["cidr-services"])
        self.service_api = IPv4Address(kv["addresses"]["service-api"])
        self.service_dns = IPv4Address(kv["addresses"]["service-dns"])

        if self.service_api not in self.cidr_services or self.service_dns not in self.cidr_services:
            command.fail("in config: expected service IPs to be in the correct CIDR")

        self.dns_upstreams = [IPv4Address(server) for server in kv["dns-upstreams"]]
        self.dns_bootstrap = {hostname: IPv4Address(ip) for hostname, ip in kv["dns-bootstrap"].items()}
        self.root_admins = kv["root-admins"]
        self.nodes = [Node(n) for n in kv["nodes"]]

        self.keyserver = None

        for node in self.nodes:
            if node.kind == "supervisor":
                if self.keyserver is not None:
                    command.fail("in config: multiple supervisors not yet supported")
                self.keyserver = node

    # TODO(#371): make this configuration setting more explicit
    def is_kerberos_enabled(self):
        return len(self.root_admins) > 0

    def has_node(self, node_name: str) -> bool:
        return any(node.hostname == node_name for node in self.nodes)

    def get_node(self, node_name: str) -> Node:
        for node in self.nodes:
            if node.hostname == node_name:
                return node
        command.fail("no such node: %s" % node_name)

    def get_any_node(self, kind: str) -> Node:
        for node in self.nodes:
            if node.kind == kind:
                return node
        command.fail("cannot find any nodes of kind %s" % kind)

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


@command.wrap
def populate() -> None:
    "initialize the cluster's setup.yaml with the template"
    setup_yaml = os.path.join(get_project(create_dir_if_missing=True), "setup.yaml")
    if os.path.exists(setup_yaml):
        command.fail("setup.yaml already exists")
    resource.copy_to("setup.yaml", setup_yaml)
    print("filled out setup.yaml")


@command.wrap
def edit() -> None:
    "open $EDITOR (defaults to nano) to edit the project's setup.yaml"
    setup_yaml = os.path.join(get_project(), "setup.yaml")
    if not os.path.exists(setup_yaml):
        command.fail("setup.yaml does not exist (run spire config populate first?)")
    subprocess.check_call([get_editor(), "--", setup_yaml])


@command.wrap
def print_cluster_conf() -> None:
    "display the generated cluster.conf"
    print(get_cluster_conf())


@command.wrap
def print_local_kubeconfig() -> None:
    "display the generated local kubeconfig"
    print(get_local_kubeconfig())


@command.wrap
def print_prometheus_yaml() -> None:
    "display the generated prometheus.yaml"
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


def get_single_kube_spec(name: str, extra_kvs: dict=None) -> str:
    templ = resource.get_resource("clustered/%s" % name).decode()
    return template.yaml_template(templ, get_kube_spec_vars(extra_kvs))


main_command = command.Mux("commands about cluster configuration", {
    "populate": populate,
    "edit": edit,
    "show": command.Mux("commands about showing different aspects of the configuration", {
        "cluster.conf": print_cluster_conf,
        "kubeconfig": print_local_kubeconfig,
        "prometheus.yaml": print_prometheus_yaml,
    }),
})
