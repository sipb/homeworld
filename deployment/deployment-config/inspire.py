#!/usr/bin/env python3

import configure
import subprocess
import time
import sys

def operation_ssh(config, node, script):
    cmd = ["echo", "ssh", "root@%s.%s" % (node.hostname, config.external_domain), script]
    return lambda: subprocess.check_call(cmd)

def operation_scp_upload(config, node, source, dest):
    cmd = ["echo", "scp", source, "root@%s.%s:%s" % (node.hostname, config.external_domain, dest)]
    return lambda: subprocess.check_call(cmd)

def operation_sleep(delay):
    return lambda: time.sleep(delay)

def generate_operations_for_install(config):
    ops = []
    cluster = [node for node in config.nodes if node.kind != "supervisor"]
    for node in cluster:
    for node in config.nodes:
        cmd = "apt-get update && apt-get upgrade -y"
        if node.kind != "supervisor":
            cmd += " && apt-get install -y homeworld-services"
        ops.append(("install packages on %s" % node.hostname, operation_ssh(config, node, cmd)))
    return ops

def generate_operations_for_admit_keyserver(config):
    ops = []
    cluster = [node for node in config.nodes if node.kind == "supervisor"]
    for node in cluster:
        ops.append(("request bootstrap token for %s" % node.hostname, operation_ssh(config, node, "/usr/bin/keyinitadmit /etc/homeworld/config/keyserver.yaml %s.%s bootstrap-keyinit >/etc/homeworld/keyclient/bootstrap.token" % (node.hostname, config.external_domain))))
    return ops

def generate_operations_for_setup_gateway(config):
    ops = []
    cluster = [node for node in config.nodes if node.kind == "supervisor"]
    for node in cluster:
        ops.append(("upload keytab for %s" % node.hostname, operation_scp_upload(config, node, "keytab-%s", "/etc/krb5.keytab")))
        ops.append(("restart gateway on %s" % node.hostname, operation_ssh(config, node, "systemctl restart keygateway")))
    return ops

def generate_operations_for_keyinit(config):
    ops = []
    cluster = [node for node in config.nodes if node.kind == "supervisor"]
    for node in cluster:
        ops.append(("upload authorities to %s" % node.hostname, operation_scp_upload(config, node, "authorities.tgz", "/etc/homeworld/keyserver/authorities/authorities.tgz")))
        ops.append(("extract authorities on %s" % node.hostname, operation_ssh(config, node, "cd /etc/homeworld/keyserver/authorities/ && tar -xzf authorities.tgz && rm authorities.tgz")))
        ops.append(("upload cluster config to %s" % node.hostname, operation_scp_upload(config, node, "confgen/cluster.conf", "/etc/homeworld/keyserver/static/cluster.conf")))
        ops.append(("upload machine list to %s" % node.hostname, operation_scp_upload(config, node, "confgen/machine.list", "/etc/homeworld/keyserver/static/machine.list")))
        ops.append(("upload keyserver config to %s" % node.hostname, operation_scp_upload(config, node, "confgen/keyserver.yaml", "/etc/homeworld/keyserver/config/keyserver.yaml")))
        ops.append(("start keyserver on %s" % node.hostname, operation_ssh(config, node, "systemctl restart keyserver.service")))
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

def generate_operations_for_start(config):
    ops = []
    masters = [node for node in config.nodes if node.kind == "master"]
    workers = [node for node in config.nodes if node.kind == "worker"]
    for master in masters:
        ops.append(("start etcd on master %s" % master.hostname, operation_ssh(config, master, "/usr/lib/hyades/start-master-etcd.sh")))
    ops.append(("pause before continuing deployment", operation_sleep(2)))
    for master in masters:
        ops.append(("start all on master %s" % master.hostname, operation_ssh(config, master, "/usr/lib/hyades/start-master.sh")))
    ops.append(("pause before continuing deployment", operation_sleep(2)))
    for worker in workers:
        ops.append(("start all on worker %s" % worker.hostname, operation_ssh(config, worker, "/usr/lib/hyades/start-worker.sh")))
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
    print("usage: inspire.py start-services", file=sys.stderr)
    print("usage: inspire.py install-packages", file=sys.stderr)
    print("usage: inspire.py bootstrap-dns", file=sys.stderr)
    print("usage: inspire.py restore-dns", file=sys.stderr)
    sys.exit(1)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        usage()
    config = configure.load_setup()
    if sys.argv[1] == "start-services":
        ops = generate_operations_for_start(config)
    elif sys.argv[1] == "install-packages":
        ops = generate_operations_for_install(config)
    elif sys.argv[1] == "deploy-keyinit":
        ops = generate_operations_for_keyinit(config)
    elif sys.argv[1] == "admit-keyserver":
        ops = generate_operations_for_admitinit(config)
    elif sys.argv[1] == "setup-keygateway":
        ops = generate_operations_for_setup_gateway(config)
    elif sys.argv[1] == "bootstrap-dns":
        ops = generate_operations_for_dns(config, True)
    elif sys.argv[1] == "restore-dns":
        ops = generate_operations_for_dns(config, False)
    else:
        usage()
    run_operations(ops)
