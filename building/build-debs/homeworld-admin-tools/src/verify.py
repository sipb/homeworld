import query
import time
import threading
import tempfile
import os
import setup
import subprocess
import command
import configuration
import access
import parallel
import random
import json


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


def check_ssh(node, command):
    config = configuration.get_config()
    subprocess.check_call(["ssh", "-o", "StrictHostKeyChecking=yes", "root@%s.%s" % (node.hostname, config.external_domain), "--"] + command)


def check_ssh_output(node, command):
    config = configuration.get_config()
    return subprocess.check_output(["ssh", "-o", "StrictHostKeyChecking=yes", "root@%s.%s" % (node.hostname, config.external_domain), "--"] + command)


def check_online(server=None):
    config = configuration.get_config()
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
            is_online = (check_ssh_output(node, ["echo", "round-trip"]) == b"round-trip\n")
        except subprocess.CalledProcessError:
            is_online = False
        if not is_online:
            any_offline = True
        print("NODE:", node.hostname.ljust(30), ("[ONLINE]" if is_online else "[OFFLINE]").rjust(10))
    if any_offline:
        command.fail("not all nodes were online!")
    print("All nodes: [ONLINE]")


def check_keygateway():
    try:
        access.call_keyreq("check")
    except subprocess.CalledProcessError as e:
        command.fail("keygateway check failed: %s" % e)
    print("keygateway access confirmed.")


def check_ssh_with_certs(hostname=None):
    config = configuration.get_config()
    if hostname is None:
        if config.keyserver is None:
            command.fail("no keyserver found")
        hostname = config.keyserver.hostname
    env = dict(os.environ)
    if "SSH_AUTH_SOCK" in env:
        del env["SSH_AUTH_SOCK"]
    if "SSH_AGENT_PID" in env:
        del env["SSH_AGENT_PID"]
    keypath = access.renew_ssh_cert()
    try:
        result = subprocess.check_output(["ssh", "-o", "StrictHostKeyChecking=yes", "-i", keypath, "root@%s.%s" % (hostname, config.external_domain), "echo confirmed"], env=env)
    except subprocess.CalledProcessError as e:
        command.fail("ssh check failed: %s" % e)
    if result != b"confirmed\n":
        command.fail("unexpected result from ssh check: '%s'" % result.decode())
    print("ssh access confirmed!")


def check_etcd_health():
    config = configuration.get_config()
    result = access.call_etcdctl(["cluster-health"], return_result=True)
    lines = result.strip().decode().split("\n")
    if lines.pop() != "cluster is healthy":
        command.fail("cluster did not report as healthy!")
    member_ids = []
    for line in lines:
        if not line.startswith("member "):
            command.fail("unexpected format of cluster-health result; perhaps the cluster is not completely healthy?")
        if "//" not in line:
            command.fail("could not find URL in cluster-health result; perhaps the cluster is not completely healthy?")
        server_name = line.split("//", 1)
        if " is unhealthy: " in line:
            command.fail("member found to be unhealthy: %s" % server_name)
        if " is unreachable: " in line:
            command.fail("member found to be unreachable: %s" % server_name)
        if " is healthy: got healthy result from https://" not in line:
            command.fail("did not find expected healthy result info for: %s" % server_name)
        member_ids.append(line.split(" ")[1])

    result = access.call_etcdctl(["member", "list"], return_result=True).decode()
    found_member_ids = []
    servers = []
    leader_count = 0
    for line in result.strip().split("\n"):
        if ": " not in line:
            command.fail("invalid member list line format")
        mid, kvs = line.split(": ", 1)
        if mid not in member_ids:
            command.fail("member id missing: %s" % mid)
        found_member_ids.append(mid)

        kvs = dict(kv.split("=", 1) for kv in kvs.split(" "))

        if set(kvs.keys()) != {"name", "peerURLs", "clientURLs", "isLeader"}:
            command.fail("unexpected format of member list (wrong set of keys?)")
        servers.append(kvs["name"])
        if kvs["isLeader"] == "true":
            leader_count += 1
        elif kvs["isLeader"] != "false":
            command.fail("unexpected value of isLeader")

    if leader_count != 1:
        command.fail("wrong number of leaders")

    if sorted(servers) != sorted(node.hostname for node in config.nodes if node.kind == "master"):
        command.fail("invalid detected set of servers: %s" % servers)

    if member_ids != found_member_ids:
        command.fail("member id list mismatch")

    print("etcd seems to be healthy!")


def get_kubectl_json(*params: str):
    raw = access.call_kubectl(list(params) + ["-o", "json"], return_result=True)
    return json.loads(raw.decode())


def check_kube_health():
    expected_kubernetes_version = "v1.8.0"
    config = configuration.get_config()

    # verify nodes

    nodes = get_kubectl_json("get", "nodes")
    try:
        if nodes["apiVersion"] != "v1":
            command.fail("wrong API version for kubectl result")
        if nodes["kind"] != "List":
            command.fail("wrong output format from kubectl")
        nodes_remaining = {node.hostname: node for node in config.nodes if node.kind != "supervisor"}
        for node in nodes["items"]:
            if node["apiVersion"] != "v1":
                command.fail("wrong API version for kubectl result")
            if node["kind"] != "Node":
                command.fail("wrong output format from kubectl")
            nodeID = node["spec"]["externalID"]
            if nodeID not in nodes_remaining:
                command.fail("invalid or duplicate node: %s" % nodeID)
            node_obj = nodes_remaining[nodeID]
            del nodes_remaining[nodeID]
            if node_obj.kind == "master":
                if node["spec"].get("unschedulable", None) is not True:
                    command.fail("expected master node to be unschedulable")
            else:
                assert node_obj.kind == "worker"
                if node["spec"].get("unschedulable", None):
                    command.fail("expected worker node to be schedulable")
            conditions = {condobj["type"]: condobj["status"] for condobj in node["status"]["conditions"]}
            if conditions["DiskPressure"] != "False":
                command.fail("expected no disk pressure")
            if conditions["MemoryPressure"] != "False":
                command.fail("expected no memory pressure")
            if conditions["OutOfDisk"] != "False":
                command.fail("expected sufficient disk space")
            if conditions["Ready"] != "True":
                command.fail("expected kubelet to be ready")
            version = node["status"]["nodeInfo"]["kubeletVersion"].split("+")[0]
            if version != expected_kubernetes_version:
                command.fail("unexpected kubernetes version: %s (expected %s)" % (version, expected_kubernetes_version))
            if version != node["status"]["nodeInfo"]["kubeProxyVersion"].split("+")[0]:
                command.fail("mismatched kube-proxy version")
        if nodes_remaining:
            command.fail("did not see expected nodes: %s" % nodes_remaining)

    except ValueError as e:
        command.fail("failed to parse kubectl result json: %s" % e)
    except KeyError as e:
        command.fail("missing key while parsing kubectl result json: %s" % e)

    # verify namespaces

    namespaces = get_kubectl_json("get", "namespaces")
    try:
        if namespaces["apiVersion"] != "v1":
            command.fail("wrong API version for kubectl result")
        if namespaces["kind"] != "List":
            command.fail("wrong output format from kubectl")
        found = set()
        for namespace in namespaces["items"]:
            if namespace["apiVersion"] != "v1":
                command.fail("wrong API version for kubectl result")
            if namespace["kind"] != "Namespace":
                command.fail("wrong output format from kubectl")
            name = namespace["metadata"]["name"]
            if name in found:
                command.fail("duplicate namespace: %s" % name)
            found.add(name)
            if namespace["status"]["phase"] != "Active":
                command.fail("namespace was inactive: %s" % name)
        for expected in {"default", "kube-system", "kube-public"}:
            if expected not in found:
                command.fail("could not find namespace: %s" % expected)
    except ValueError as e:
        command.fail("failed to parse kubectl result json: %s" % e)
    except KeyError as e:
        command.fail("missing key while parsing kubectl result json: %s" % e)

    print("kubernetes cluster passed cursory inspection!")


def check_aci_pull():
    config = configuration.get_config()
    workers = [node for node in config.nodes if node.kind == "worker"]
    if not workers:
        command.fail("expected nonzero number of worker nodes")
    worker = random.choice(workers)
    print("trying container pulling on: %s" % worker)
    container_command = "ping -c 1 8.8.8.8 && echo 'PING RESULT SUCCESS' || echo 'PING RESULT FAIL'"
    server_command = ["rkt", "run", "--pull-policy=update", "homeworld.mit.edu/debian", "--exec", "/bin/bash", "--", "-c",
                      setup.escape_shell(container_command)]
    results = check_ssh_output(worker, server_command)
    last_line = results.replace(b"\r\n",b"\n").replace(b"\0",b'').strip().split(b"\n")[-1]
    if b"PING RESULT FAIL" in last_line:
        if b"PING RESULT SUCCESS" in last_line:
            command.fail("should not have seen both success and failure markers in last line")
        command.fail("cluster network probably not up (could not ping 8.8.8.8)")
    elif b"PING RESULT SUCCESS" not in last_line:
        command.fail("container does not seem to have launched properly; container launches are likely broken (line = %s)" % repr(last_line))
    print("container seems to be launched, with the correct network!")


def check_flannel_kubeinfo():
    config = configuration.get_config()

    # checking kubernetes info on flannel

    pods = get_kubectl_json("get", "pods", "--namespace=kube-system", "--selector=app=flannel")
    try:
        if pods["apiVersion"] != "v1":
            command.fail("FLANNEL FAILED: wrong API version for kubectl result")
        if pods["kind"] != "List":
            command.fail("FLANNEL FAILED: wrong output format from kubectl")
        found_nodes = set()
        for pod in pods["items"]:
            if pod["apiVersion"] != "v1":
                command.fail("FLANNEL FAILED: wrong API version for kubectl result")
            if pod["kind"] != "Pod":
                command.fail("FLANNEL FAILED: wrong output format from kubectl")
            name = pod["metadata"]["name"]
            if not name.startswith("kube-flannel-ds-"):
                command.fail("FLANNEL FAILED: expected kube-flannel-ds container, not %s" % name)
            node_name = pod["spec"]["nodeName"]
            if node_name in found_nodes:
                command.fail("FLANNEL FAILED: duplicate flannel on node: %s" % node_name)
            found_nodes.add(node_name)
            if pod["status"]["phase"] != "Running":
                command.fail("FLANNEL FAILED: pod was not running: %s: %s" % (name, pod["status"]["phase"]))

            conditions = {condobj["type"]: condobj["status"] for condobj in pod["status"]["conditions"]}
            if conditions["Initialized"] != "True":
                command.fail("FLANNEL FAILED: pod not yet initialized")
            if conditions["Ready"] != "True":
                command.fail("FLANNEL FAILED: pod not yet ready")

            if len(pod["status"]["containerStatuses"]) != 1:
                command.fail("FLANNEL FAILED: expected only one container")
            if pod["status"]["containerStatuses"][0]["ready"] is not True:
                command.fail("FLANNEL FAILED: expected container to be ready")

        if found_nodes != {node.hostname for node in config.nodes if node.kind != "supervisor"}:
            command.fail("FLANNEL FAILED: did not find proper set of nodes")
    except ValueError as e:
        command.fail("FLANNEL FAILED: failed to parse kubectl result json: %s" % e)
    except KeyError as e:
        command.fail("FLANNEL FAILED: missing key while parsing kubectl result json: %s" % e)

    print("flannel appears to be launched!")


def check_flannel_function():
    # checking flannel functionality
    config = configuration.get_config()

    workers = [node for node in config.nodes if node.kind == "worker"]
    if len(workers) < 2:
        command.fail("expected at least two worker nodes")
    worker_talker = random.choice(workers)
    workers.remove(worker_talker)
    worker_listener = random.choice(workers)
    assert worker_talker != worker_listener

    print("trying flannel functionality test with", worker_talker, "talking and", worker_listener, "listening")
    print("checking launch on both systems...")

    # this is here to make sure both servers have pulled the relevant containers
    server_command = ["rkt", "run", "--net=rkt.kubernetes.io", "homeworld.mit.edu/debian", "--", "-c", "/bin/true"]
    for worker in (worker_talker, worker_listener):
        check_ssh(worker, server_command)

    print("ready -- this may take a minute... please be patient")

    found_address = [None]
    event = threading.Event()

    def listen():
        try:
            container_command = "ip -o addr show dev eth0 to 172.18/16 primary && sleep 15"
            server_command = ["rkt", "run", "--net=rkt.kubernetes.io", "homeworld.mit.edu/debian", "--", "-c", setup.escape_shell(container_command)]
            cmd = ["ssh", "-o", "StrictHostKeyChecking=yes", "root@%s.%s" % (worker_listener.hostname, config.external_domain), "--"] + server_command
            with subprocess.Popen(cmd, stdout=subprocess.PIPE, bufsize=1, universal_newlines=True) as process:
                stdout = process.stdout.readline()
                if "scope" not in stdout:
                    command.fail("could not find scope line in ip addr output (%s)" % repr(stdout))
                parts = stdout.split(" ")
                if "inet" not in parts:
                    command.fail("could not find inet address in ip addr output")
                address = parts[parts.index("inet") + 1]
                if not address.endswith("/24"):
                    command.fail("expected address that ended in /24, not '%s'" % address)
                address = address[:-3]
                if address.count(".") != 3:
                    command.fail("expected valid IPv4 address, not '%s'" % address)
                if not address.replace(".", "").isdigit():
                    command.fail("expected valid IPv4 address, not '%s'" % address)
                found_address[0] = address
                event.set()
                process.communicate(timeout=20)
        finally:
            event.set()
        return True

    def talk():
        if not event.wait(25):
            command.fail("timed out while waiting for IPv4 address of listener")
        address = found_address[0]
        if address is None:
            command.fail("no address was specified by listener")
        container_command = "ping -c 1 %s && echo 'PING RESULT SUCCESS' || echo 'PING RESULT FAIL'" % address
        server_command = ["rkt", "run", "--net=rkt.kubernetes.io", "homeworld.mit.edu/debian", "--exec", "/bin/bash", "--", "-c", setup.escape_shell(container_command)]
        results = check_ssh_output(worker_talker, server_command)
        last_line = results.replace(b"\r\n",b"\n").replace(b"\0",b'').strip().split(b"\n")[-1]
        if b"PING RESULT FAIL" in last_line:
            command.fail("was not able to ping the target container; is flannel working?")
        elif b"PING RESULT SUCCESS" not in last_line:
            command.fail("could not launch container to test flannel properly")
        return True

    passed_1, passed_2 = parallel.parallel(listen, talk)
    assert passed_1 is True and passed_2 is True, "should have been checked by parallel already!"

    print("flannel seems to work!")


def check_dns_kubeinfo():
    config = configuration.get_config()

    pods = get_kubectl_json("get", "pods", "--namespace=kube-system", "--selector=k8s-app=kube-dns")
    try:
        if pods["apiVersion"] != "v1":
            command.fail("wrong API version for kubectl result")
        if pods["kind"] != "List":
            command.fail("wrong output format from kubectl")
        allowed_nodes = {node.hostname for node in config.nodes if node.kind == "worker"}
        for pod in pods["items"]:
            if pod["apiVersion"] != "v1":
                command.fail("wrong API version for kubectl result")
            if pod["kind"] != "Pod":
                command.fail("wrong output format from kubectl")
            name = pod["metadata"]["name"]
            if not name.startswith("kube-dns-v20-"):
                command.fail("expected kube-dns-v20 container, not %s" % name)
            node_name = pod["spec"]["nodeName"]
            if node_name not in allowed_nodes:
                command.fail("dns-addon not running on acceptable node: %s" % node_name)
            if pod["status"]["phase"] != "Running":
                command.fail("pod was not running: %s: %s" % (name, pod["status"]["phase"]))

            conditions = {condobj["type"]: condobj["status"] for condobj in pod["status"]["conditions"]}
            if conditions["Initialized"] != "True":
                command.fail("pod not yet initialized")
            if conditions["Ready"] != "True":
                command.fail("pod not yet ready")
            if conditions["PodScheduled"] != "True":
                command.fail("pod not yet scheduled")

            if len(pod["status"]["containerStatuses"]) != 3:
                command.fail("expected three containers")
            for container in pod["status"]["containerStatuses"]:
                if container["ready"] is not True:
                    command.fail("expected container to be ready")
    except ValueError as e:
        command.fail("failed to parse kubectl result json: %s" % e)
    except KeyError as e:
        command.fail("missing key while parsing kubectl result json: %s" % e)

    print("dns-addon appears to be launched!")


def check_dns_function():
    config = configuration.get_config()

    workers = [node for node in config.nodes if node.kind == "worker"]
    if len(workers) < 1:
        command.fail("expected at least one worker node")
    worker = random.choice(workers)

    print("trying dns functionality test with", worker)

    container_command = "nslookup kubernetes.default.svc.hyades.local 172.28.0.2"
    server_command = ["rkt", "run", "homeworld.mit.edu/debian", "--exec", "/bin/bash", "--", "-c", setup.escape_shell(container_command)]
    results = check_ssh_output(worker, server_command)
    last_line = results.replace(b"\r\n",b"\n").replace(b"\0",b'').strip().split(b"\n")[-1]
    if not last_line.endswith(b"Address: 172.28.0.1"):
        command.fail("unexpected last line: %s" % repr(last_line.decode()))

    print("dns-addon seems to work!")


main_command = command.mux_map("commands about verifying the state of a cluster", {
    "keystatics": command.wrap("verify that keyserver static files are being served properly", check_keystatics),
    "keygateway": command.wrap("verify that the keygateway has been properly started", check_keygateway),
    "online": command.wrap("check whether a server (or all servers) is/are accepting SSH connections", check_online),
    "ssh-with-certs": command.wrap("check if certificate-based SSH access works", check_ssh_with_certs),
    "etcd": command.wrap("verify that etcd is healthy and working", check_etcd_health),
    "kubernetes": command.wrap("verify that kubernetes appears healthy", check_kube_health),
    "aci-pull": command.wrap("verify that rkt can (or was able to) pull containers from the homeworld registry", check_aci_pull),
    "flannel-run": command.wrap("verify that kubernetes has launched flannel successfully", check_flannel_kubeinfo),
    "flannel-ping": command.wrap("verify that flannel-enabled containers on different hosts can talk", check_flannel_function),
    "dns-addon-run": command.wrap("verify that kubernetes has launched dns-addon successfully", check_dns_kubeinfo),
    "dns-addon-query": command.wrap("verify that the DNS addon is successfully handling requests", check_dns_function),
})
