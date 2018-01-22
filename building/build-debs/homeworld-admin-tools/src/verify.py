import query
import requests
import urllib
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
import ssh


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


def pull_prometheus_query(query):
    config = configuration.get_config()
    host_options = [node.hostname for node in config.nodes if node.kind == "supervisor"]
    if len(host_options) < 1:
        command.fail("expected at least one supervisor node")
    url = "http://%s.%s:9090/api/v1/query?%s" % (host_options[0], config.external_domain, urllib.parse.urlencode({"query": query}))
    resp = requests.get(url)
    resp.raise_for_status()
    body = resp.json()
    if type(body) != dict or body.get("status") != "success" or type(body.get("data")) != dict:
        command.fail("prometheus query failed")
    data = body["data"]
    if data.get("resultType") != "vector":
        command.fail("prometheus query did not produce a vector")
    result_vec = data["result"]
    if type(result_vec) != list or len(result_vec) != 1:
        command.fail("prometheus query returned %d results instead of 1" % len(result_vec))
    if type(result_vec[0]) != dict or "value" not in result_vec[0] or len(result_vec[0]["value"]) != 2:
        command.fail("unexpected format of result")
    return result_vec[0]["value"][1]


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


def check_keygateway():
    access.call_keyreq("check")
    print("keygateway access confirmed.")


def check_ssh_with_certs(hostname=None):
    config = configuration.get_config()
    if hostname is None:
        if config.keyserver is None:
            command.fail("no keyserver found")
        node = config.keyserver
    else:
        node = config.get_node(hostname)
    env = dict(os.environ)
    if "SSH_AUTH_SOCK" in env:
        del env["SSH_AUTH_SOCK"]
    if "SSH_AGENT_PID" in env:
        del env["SSH_AGENT_PID"]
    keypath = access.renew_ssh_cert()
    try:
        result = subprocess.check_output(ssh.SSH_BASE + ["-i", keypath, ssh.ssh_get_login(node), "echo", "confirmed"], env=env)
    except subprocess.CalledProcessError as e:
        command.fail("ssh check failed: %s" % e)
    if result != b"confirmed\n":
        command.fail("unexpected result from ssh check: '%s'" % result.decode())
    print("ssh access confirmed!")


def check_online():
    if not int(pull_prometheus_query("verify_infra_nodes_online")):
        command.fail("prometheus signaled that not all nodes are online")
    if not int(pull_prometheus_query("verify_infra_nodes_accessible")):
        command.fail("prometheus signaled that not all nodes are configured and accessible with SSH")
    print("basic infrastructure seems to be healthy!")


def check_etcd_health():
    if not int(pull_prometheus_query("verify_etcd_all")):
        command.fail("prometheus signaled unhealthy etcd cluster")
    print("etcd seems to be healthy!")


def check_kube_init():
    if not int(pull_prometheus_query("verify_kube_apiservers_up")):
        command.fail("prometheus signaled lack of kubernetes cluster")
    print("kubernetes cluster passed first-stage inspection!")


def check_kube_health():
    if not int(pull_prometheus_query("verify_kube_all")):
        command.fail("prometheus signaled incomplete kubernetes cluster")
    print("kubernetes cluster passed second-stage inspection!")


def check_aci_pull():
    if not int(pull_prometheus_query("verify_aci_all")):
        command.fail("prometheus signaled failure of aci pulling")
    print("aci pulling seems to work!")


def check_flannel():
    if not int(pull_prometheus_query("verify_flannel_all")):
        command.fail("prometheus signaled failure of flannel")
    print("flannel seems to work!")


def check_dns():
    if not int(pull_prometheus_query("verify_dns_all")):
        command.fail("prometheus signaled failure of dns")
    print("dns-addon seems to work!")


main_command = command.mux_map("commands about verifying the state of a cluster", {
    "keystatics": command.wrap("verify that keyserver static files are being served properly", check_keystatics),
    "keygateway": command.wrap("verify that the keygateway has been properly started", check_keygateway),
    "online": command.wrap("check whether a server (or all servers) is/are accepting SSH connections", check_online),
    "ssh-with-certs": command.wrap("check if certificate-based SSH access works", check_ssh_with_certs),
    "etcd": command.wrap("verify that etcd is healthy and working", check_etcd_health),
    "kubernetes-init": command.wrap("verify that kubernetes appears initialized", check_kube_init),
    "kubernetes": command.wrap("verify that kubernetes appears healthy", check_kube_health),
    "aci-pull": command.wrap("verify that aci pulling from the homeworld registry, and associated container execution, are functioning", check_aci_pull),
    "flannel": command.wrap("verify that the flannel addon is functioning", check_flannel),
    "dns-addon": command.wrap("verify that the DNS addon is functioning", check_dns),
})
