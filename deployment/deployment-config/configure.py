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
