#!/usr/bin/env python3

# may require: pip3 install --user building/upstream/PyYAML-3.12.tar.gz

import yaml
import os

class CIDR:
    def __init__(self, cidr):
        if cidr.count("/") != 1:
            raise Exception("invalid cidr: %s" % cidr)
        ip, bits = cidr.split("/")
        bits = int(bits)
        if bits < 0 or bits > 32:
            raise Exception("invalid bits: %s" % bits)
        self.ip = IP(ip)
        self.bits = bits
        self.cidr = "%s/%d" % (self.ip.ip, self.bits)

    def netmask(self):
        return IP.from_integer(((1 << (32 - self.bits)) - 1) ^ 0xFFFFFFFF)

    def __contains__(self, ip):
        return (ip & self.netmask()) == self.ip

    def __str__(self):
        return self.cidr

    def __repr__(self):
        return "cidr:" + self.cidr

class IP:
    def __init__(self, ip):
        if ip.count(".") != 3:
            raise Exception("invalid ipv4 address: %s" % ip)
        self.octets = [int(x) for x in ip.split(".")]
        for octet in self.octets:
            if octet < 0 or octet >= 256:
                raise Exception("invalid ipv4 address: bad octet %s" % octet)
        self.ip = "%d.%d.%d.%d" % tuple(self.octets)

    @classmethod
    def from_integer(cls, num):
        assert 0 <= num <= 0xFFFFFFFF
        octets = (((num & 0xFF000000) >> 24),
                  ((num & 0x00FF0000) >> 16),
                  ((num & 0x0000FF00) >> 8),
                  ((num & 0x000000FF) >> 0))
        return IP("%d.%d.%d.%d" % octets)

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

class Config:
    pass

class Node:
    def __init__(self, config):
        if set(config.keys()) != {"hostname", "ip", "kind"}:
            raise Exception("invalid sections in node configuration")
        self.hostname = config["hostname"]
        self.ip = IP(config["ip"])
        self.kind = config["kind"]
        if self.kind not in {"master", "worker", "supervisor"}:
            raise Exception("invalid node kind: %s" % self.kind)

    def __repr__(self):
        return "%s node %s (%s)" % (self.kind, self.hostname, self.ip)

class Template:
    def __init__(self, filename, load=True):
        if load:
            with open(filename, "r") as f:
                self._template = f.read().split("\n")
        else:
            self._template = filename.split("\n")

    def visible_lines(self, keys):
        for line in self._template:
            if line.startswith("[") and "]" in line:
                condition, line = line[1:].split("]", 1)
                if not keys[condition]:
                    continue
            yield line + "\n"

    def template(self, keys):
        fragments = []
        for line in self.visible_lines(keys):
            while "{{" in line:
                prefix, rest = line.split("{{", 1)
                if "}}" in prefix:
                    raise Exception("unbalanced substitution")
                fragments.append(prefix)
                if "}}" not in rest:
                    raise Exception("unbalanced substitution")
                key, line = rest.split("}}", 1)
                fragments.append(str(keys[key]))
            if "}}" in line:
                raise Exception("unbalanced substitution")
            fragments.append(line)
        return "".join(fragments)

def load_setup():
    with open("setup.yaml", "r") as f:
        config = yaml.safe_load(f)

    if set(config.keys()) != {"cluster", "addresses", "dns-bootstrap", "root-admins", "nodes"}:
        raise Exception("invalid sections in configuration file")

    cobj = Config()

    cluster = config["cluster"]
    if set(cluster.keys()) != {"external-domain", "internal-domain", "etcd-token"}:
        raise Exception("invalid keys in cluster configuration section")
    assert all(type(v) == str for v in cluster.values())
    cobj.external_domain = cluster["external-domain"]
    cobj.internal_domain = cluster["internal-domain"]
    cobj.etcd_token = cluster["etcd-token"]

    addresses = config["addresses"]
    if set(addresses.keys()) != {"cidr-pods", "cidr-services", "service-api", "service-dns"}:
        raise Exception("invalid keys in addresses configuration section")
    assert all(type(v) == str for v in addresses.values())
    cobj.cidr_pods = CIDR(addresses["cidr-pods"])
    cobj.cidr_services = CIDR(addresses["cidr-services"])
    cobj.service_api = IP(addresses["service-api"])
    cobj.service_dns = IP(addresses["service-dns"])

    if cobj.service_api not in cobj.cidr_services or cobj.service_dns not in cobj.cidr_services:
        raise Exception("expected services to be in the correct CIDR")

    cobj.dns_bootstrap = {hostname: IP(ip) for hostname, ip in config["dns-bootstrap"].items()}

    cobj.root_admins = config["root-admins"]
    assert all(type(v) == str for v in cobj.root_admins)

    cobj.nodes = [Node(n) for n in config["nodes"]]

    compute_details(cobj)

    return cobj

def compute_details(config):
    config.keyserver = None

    for node in config.nodes:
        if node.kind == "supervisor":
            if config.keyserver is not None:
                raise Exception("multiple supervisors not yet supported")
            config.keyserver = node

def generate_results(config, target_dir):
    if not os.path.isdir(target_dir):
        os.mkdir(target_dir)

    kc = Template("keyclient.yaml.in")
    ks = Template("keyserver.yaml.in")

    for k,v in config.__dict__.items():
        print(k,"=>",v)

    for node in ["master", "worker", "base", "supervisor"]:
        tpl = {"KEYSERVER": config.keyserver.hostname + "." + config.external_domain,
                  "MASTER": node == "master",
                  "WORKER": node in ("worker", "master"),
                    "BASE": node == "base"}
        result = kc.template(tpl)
        with open(os.path.join(target_dir, "keyclient-%s.yaml" % node), "w") as f:
            f.write(result)

    accounts = []

    nodetemplate = Template(
"""  - principal: {{HOSTNAME}}.{{DOMAIN}}
    group: {{KIND}}-nodes
    limit-ip: true
    metadata:
      ip: {{IP}}
      hostname: {{HOSTNAME}}
      schedule: {{SCHEDULE}}
      kind: {{KIND}}""", load=False)

    for node in config.nodes:
        tvs = {"HOSTNAME": node.hostname,
                 "DOMAIN": config.external_domain,
                     "IP": node.ip,
                   "KIND": node.kind,
               "SCHEDULE": "true" if node.kind == "worker" else "false"}
        accounts.append(nodetemplate.template(tvs))

    admintemplate = Template(
"""  - principal: {{PRINCIPAL}}
    disable-direct-auth: true
    group: root-admins""", load=False)

    for root_admin in config.root_admins:
        accounts.append(admintemplate.template({"PRINCIPAL": root_admin}))

    ksrv = {     "SERVICEAPI": config.service_api,
                   "ACCOUNTS": "\n".join(accounts),
                     "DOMAIN": config.external_domain,
            "INTERNAL-DOMAIN": config.internal_domain}

    result = ks.template(ksrv)
    with open(os.path.join(target_dir, "keyserver.yaml"), "w") as f:
        f.write(result)

    apiservers = [node for node in config.nodes if node.kind == "master"]

    cconf = {"APISERVER": "https://%s:443" % apiservers[0].ip,
             "APISERVER_COUNT": len(apiservers),
             "CLUSTER_CIDR": config.cidr_pods,
             "CLUSTER_DOMAIN": config.internal_domain,
             "DOMAIN": config.external_domain,
             "ETCD_CLUSTER": ",".join("%s=https://%s:2380" % (n.hostname, n.ip) for n in apiservers),
             "ETCD_ENDPOINTS": ",".join("https://%s:2379" % n.ip for n in apiservers),
             "ETCD_TOKEN": config.etcd_token,
             "SERVICE_API": config.service_api,
             "SERVICE_CIDR": config.cidr_services,
             "SERVICE_DNS": config.service_dns}

    with open(os.path.join(target_dir, "cluster.conf"), "w") as f:
        f.write("# generated by configure.py from setup.yaml\n")
        for k, v in sorted(cconf.items()):
            f.write("%s=%s\n" % (k, v))

    with open(os.path.join(target_dir, "machine.list"), "w") as f:
        f.write(",".join("%s.%s" % (node.hostname, config.external_domain) for node in config.nodes) + "\n")

def generate_cluster_config(config, source_dir, target_dir):
    if not os.path.isdir(target_dir):
        os.mkdir(target_dir)
    vars = {"NETWORK": config.cidr_pods}
    for config in os.listdir(source_dir):
        source = os.path.join(source_dir, config)
        target = os.path.join(target_dir, config)
        template = Template(source)
        templated = template.template(vars)
        with open(target, "w") as f:
            f.write(templated)

if __name__ == "__main__":
    config = load_setup()
    generate_results(config, "./confgen/")
