#!/usr/bin/env python3

import configure
import subprocess
import time
import sys

def operation_ssh(config, node, script):
    cmd = ["ssh", "root@%s.%s" % (node.hostname, config.external_domain), script]
    return lambda: subprocess.check_call(cmd)

def operation_scp_upload(config, node, source, dest):
    cmd = ["scp", source, "root@%s.%s:%s" % (node.hostname, config.external_domain, dest)]
    return lambda: subprocess.check_call(cmd)

def operation_sleep(delay):
    return lambda: time.sleep(delay)

def generate_operations_for_setup_bootstrap_registry(config):
    ops = []
    cluster = [node for node in config.nodes if node.kind == "supervisor"]
    for node in cluster:
        ops.append(("create ssl cert directory on %s" % node.hostname, operation_ssh(config, node, "mkdir -p /etc/homeworld/ssl")))
        ops.append(("upload homeworld.mit.edu key for %s" % node.hostname, operation_scp_upload(config, node, "ssl/homeworld.mit.edu.key", "/etc/homeworld/ssl/homeworld.mit.edu.key")))
        ops.append(("upload homeworld.mit.edu cert for %s" % node.hostname, operation_scp_upload(config, node, "ssl/homeworld.mit.edu.pem", "/etc/homeworld/ssl/homeworld.mit.edu.pem")))
        ops.append(("restart nginx on %s" % node.hostname, operation_ssh(config, node, "systemctl restart nginx")))
    return ops

def generate_operations_for_dns(config, is_install):
    ops = []
    cluster = [node for node in config.nodes if node.kind != "supervisor"]
    for node in cluster:
        cmd = "grep -vF AUTO-HOMEWORLD-BOOTSTRAP /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts"
        ops.append(("strip bootstrapped dns on %s" % node.hostname, operation_ssh(config, node, cmd)))
        if is_install:
            for hostname, ip in config.dns_bootstrap.items():
                cmd = "echo '%s\t%s # AUTO-HOMEWORLD-BOOTSTRAP' >>/etc/hosts" % (ip, hostname)
                ops.append(("bootstrap dns on %s: %s" % (node.hostname, hostname), operation_ssh(config, node, cmd)))
    return ops

def run_operations(ops):
    print("== executing %d operations ==" % len(ops))
    print()
    startat = time.time()
    for i, (name, operation) in enumerate(ops, 1):
        print("--", name, " (%d/%d) --" % (i, len(ops)))
        operation()
        print()
    print("== all operations executed in %.2f seconds! ==" % (time.time() - startat))

def usage():
    print("usage: inspire.py setup-bootstrap-registry", file=sys.stderr)
    print("usage: inspire.py bootstrap-dns", file=sys.stderr)
    print("usage: inspire.py restore-dns", file=sys.stderr)
    sys.exit(1)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        usage()
    config = configure.load_setup()
    if sys.argv[1] == "just-supervisor":
        config.nodes = [node for node in config.nodes if node.kind == "supervisor"]
        del sys.argv[1]
    if len(sys.argv) != 2:
        usage()
    if sys.argv[1] == "setup-bootstrap-registry":
        ops = generate_operations_for_setup_bootstrap_registry(config)
    elif sys.argv[1] == "bootstrap-dns":
        ops = generate_operations_for_dns(config, True)
    elif sys.argv[1] == "restore-dns":
        ops = generate_operations_for_dns(config, False)
    else:
        usage()
    run_operations(ops)
