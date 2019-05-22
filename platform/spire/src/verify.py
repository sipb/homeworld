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
    if type(result_vec) != list or len(result_vec) > 1:
        command.fail("prometheus query returned %d results instead of 1" % len(result_vec))
    if not result_vec:
        command.fail("no results from prometheus query '%s'" % query)
    if type(result_vec[0]) != dict or "value" not in result_vec[0] or len(result_vec[0]["value"]) != 2:
        command.fail("unexpected format of result")
    return result_vec[0]["value"][1]


def check_keystatics():
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


def check_certs_on_supervisor():
    config = configuration.get_config()
    for node in config.nodes:
        if node.kind == "supervisor":
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/authorities/kubernetes.pem")
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/keys/kubernetes-worker.pem")
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/ssl/homeworld.private.pem")


def expect_prometheus_query_exact(query, expected, description):  # description -> 'X are Y'
    count = int(pull_prometheus_query(query))
    if count > expected:
        command.fail("too many %s" % description)
    if count < expected:
        command.fail("only %d/%d %s" % (count, expected, description))


def expect_prometheus_query_bool(query, message):
    if not int(pull_prometheus_query(query)):
        command.fail(message)


def check_online():
    config = configuration.get_config()
    nodes_expected = len(config.nodes)
    expect_prometheus_query_exact('sum(up{job="node-resources"})', nodes_expected, "nodes are online")
    expect_prometheus_query_exact('sum(keysystem_ssh_access_check)', nodes_expected, "nodes are accessible")
    print("all", nodes_expected, "nodes are online and accessible")


def check_etcd_health():
    config = configuration.get_config()
    master_node_count = len([node for node in config.nodes if node.kind == "master"])
    expect_prometheus_query_exact('sum(etcd_server_has_leader)', master_node_count, "etcd servers are online")
    if float(pull_prometheus_query('sum(rate(etcd_server_proposals_committed_total[1m]))')) < 1:
        command.fail("etcd is not committing any proposals; is likely not healthy")
    print("all", master_node_count, "etcd servers seems to be healthy!")


def check_kube_init():
    config = configuration.get_config()
    master_node_count = len([node for node in config.nodes if node.kind == "master"])
    expect_prometheus_query_exact('sum(up{job="kubernetes-apiservers"})', master_node_count, "kubernetes apiservers are online")
    print("all", master_node_count, "kubernetes apiservers seem to be online!")


def check_kube_health():
    check_kube_init()
    config = configuration.get_config()
    kube_node_count = len([node for node in config.nodes if node.kind != "supervisor"])
    master_node_count = len([node for node in config.nodes if node.kind == "master"])
    expect_prometheus_query_exact('sum(kube_node_info)', kube_node_count, "kubernetes nodes are online")

    hostnames = [node.hostname for node in config.nodes if node.kind == "master"]
    regex_for_master_nodes = "|".join(hostnames)
    for hostname in hostnames:
        if not hostname.replace("-", "").isalnum():
            command.fail("invalid hostname for inclusion in prometheus monitoring rules: %s" % hostname)
    expect_prometheus_query_exact('sum(kube_node_spec_unschedulable{node=~"%s"})' % regex_for_master_nodes,
                                  master_node_count, "master nodes are unschedulable")
    expect_prometheus_query_exact('sum(kube_node_spec_unschedulable)',
                                  master_node_count, "kubernetes nodes are unschedulable")
    expect_prometheus_query_exact('sum(kube_node_status_condition{condition="Ready",status="true"})',
                                  kube_node_count, "kubernetes nodes are ready")
    NAMESPACES = ["default", "kube-public", "kube-system"]
    expect_prometheus_query_exact('sum(kube_namespace_status_phase{phase="Active",namespace=~"%s"})' % "|".join(NAMESPACES),
                                  len(NAMESPACES), "namespaces are set up")
    print("kubernetes cluster passed cursory inspection!")


def check_pull():
    config = configuration.get_config()
    node_count = len([node for node in config.nodes if node.kind != "supervisor"])
    expect_prometheus_query_exact("sum(oci_pull_check)", node_count, "nodes are pulling ocis properly")
    expect_prometheus_query_exact("sum(oci_exec_check)", node_count, "nodes are launching ocis properly")
    print("oci pulling seems to work!")


def check_flannel():
    config = configuration.get_config()
    node_count = len([node for node in config.nodes if node.kind != "supervisor"])
    expect_prometheus_query_exact('sum(kube_daemonset_status_number_ready{daemonset="kube-flannel-ds"})', node_count, "flannel pods are ready")
    expect_prometheus_query_bool("sum(flannel_collect_enum_check)", "flannel metrics collector is failing enumeration")
    expect_prometheus_query_bool("sum(flannel_collect_enum_dup_check)", "flannel metrics collector is encountering duplication")
    expect_prometheus_query_exact('sum(flannel_collect_check)', node_count, "flannel metrics monitors are collecting")
    expect_prometheus_query_exact('sum(flannel_duplicate_check)', node_count, "flannel metrics monitors are avoiding duplication")
    expect_prometheus_query_exact('sum(flannel_monitor_check)', node_count, "flannel metrics monitors are monitoring successfully")
    worst_recency = float(pull_prometheus_query('time() - min(flannel_monitor_recency)'))
    if worst_recency > 60:
        command.fail("flannel metrics monitors have not updated recently enough")
    expect_prometheus_query_exact('sum(flannel_talk_check)', node_count * node_count, "flannel pings are successful")
    print("flannel seems to work!")


def check_dns():
    ready_replicas = int(pull_prometheus_query('sum(kube_replicationcontroller_status_ready_replicas{replicationcontroller="kube-dns-v20"})'))
    spec_replicas = int(pull_prometheus_query('sum(kube_replicationcontroller_spec_replicas{replicationcontroller="kube-dns-v20"})'))
    if spec_replicas < 2:
        command.fail("not enough replicas requested by spec")
    if ready_replicas < spec_replicas - 1:  # TODO: require precise results; not currently possible due to issues with DNS containers
        command.fail("not enough DNS replicas are ready")
    if float(pull_prometheus_query('avg(dns_lookup_internal_check)')) < 1:
        command.fail("dns lookup check failed")
    if float(pull_prometheus_query('time() - min(dns_lookup_recency)')) > 30:
        command.fail("dns lookup check is not recent enough")
    print("dns-addon seems to work!")


main_command = command.mux_map("commands about verifying the state of a cluster", {
    "keystatics": command.wrap("verify that keyserver static files are being served properly", check_keystatics),
    "keygateway": command.wrap("verify that the keygateway has been properly started", check_keygateway),
    "online": command.wrap("check whether a server (or all servers) is/are accepting SSH connections", check_online),
    "ssh-with-certs": command.wrap("check if certificate-based SSH access works", check_ssh_with_certs),
    "supervisor-certs": command.wrap("verify that certificates have been uploaded to the supervisor", check_certs_on_supervisor),
    "etcd": command.wrap("verify that etcd is healthy and working", check_etcd_health),
    "kubernetes-init": command.wrap("verify that kubernetes appears initialized", check_kube_init),
    "kubernetes": command.wrap("verify that kubernetes appears healthy", check_kube_health),
    "pull": command.wrap("verify that container pulling from the homeworld registry, and associated container execution, are functioning", check_pull),
    "flannel": command.wrap("verify that the flannel addon is functioning", check_flannel),
    "dns-addon": command.wrap("verify that the DNS addon is functioning", check_dns),
})
