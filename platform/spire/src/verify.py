import json
import os
import requests
import subprocess
import tempfile
import urllib
import yaml

import access
import authority
import command
import configuration
import query
import resources
import setup
import ssh
import util


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


def pull_prometheus_query(query, default_value=None):
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
        if default_value is None:
            command.fail("no results from prometheus query '%s'" % query)
        return default_value
    if type(result_vec[0]) != dict or "value" not in result_vec[0] or len(result_vec[0]["value"]) != 2:
        command.fail("unexpected format of result")
    return result_vec[0]["value"][1]


@command.wrap
def check_keystatics():
    cluster_conf = query.get_keyurl_data("/static/cluster.conf")
    expected_cluster_conf = configuration.get_cluster_conf()

    if not compare_multiline(cluster_conf, expected_cluster_conf):
        command.fail("MISMATCH: cluster.conf")

    print("pass: keyserver serving correct static files")


@command.wrap
def check_keygateway():
    "verify that the keygateway has been properly started"
    access.call_keyreq("check")
    print("keygateway access confirmed.")


@command.wrap
def check_ssh_with_certs(hostname=None):
    "check if certificate-based SSH access works"

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


@command.wrap
def check_certs_on_supervisor():
    "verify that certificates have been uploaded to the supervisor"

    config = configuration.get_config()
    for node in config.nodes:
        if node.kind == "supervisor":
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/authorities/kubernetes.pem")
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/keys/kubernetes-supervisor.pem")
            ssh.check_ssh(node, "test", "-e", "/etc/homeworld/ssl/homeworld.private.pem")


def expect_prometheus_query_exact(query, expected, description):  # description -> 'X are Y'
    count = int(pull_prometheus_query(query))
    if count > expected:
        command.fail("too many %s" % description)
    if count < expected:
        command.fail("only %d/%d %s" % (count, expected, description))


def expect_prometheus_query_bool(query, message, accept_missing=False):
    if not int(pull_prometheus_query(query, default_value=(1 if accept_missing else None))):
        command.fail(message)


@command.wrap
def check_supervisor_accessible():
    "check whether the supervisor node is accessible over ssh"
    config = configuration.get_config()
    ssh.check_ssh(config.keyserver, "true")


@command.wrap
def check_online():
    "check whether all servers are accepting SSH connections"
    config = configuration.get_config()
    nodes_expected = len(config.nodes)
    expect_prometheus_query_exact('sum(up{job="node-resources"})', nodes_expected, "nodes are online")
    expect_prometheus_query_exact('sum(keysystem_ssh_access_check)', nodes_expected, "nodes are accessible")
    print("all", nodes_expected, "nodes are online and accessible")


@command.wrap
def check_systemd_services():
    "verify that systemd services are healthy and working"
    config = configuration.get_config()
    servicemap = yaml.safe_load(resources.get_resource("servicemap.yaml"))
    for service in servicemap["services"]:
        name = service["name"]
        kinds = service["kinds"]
        if len(kinds) == 0:
            raise Exception("must have at least one kind specified in servicemap entry for %s" % name)
        for kind in kinds:
            if kind not in configuration.Node.VALID_NODE_KINDS:
                raise Exception("unknown kind: %s" % kind)
        for node in config.nodes:
            should_run = node.kind in kinds
            state = "active" if should_run else "inactive"
            instance = "%s:9100" % node.external_dns_name()
            expect_prometheus_query_bool(
                'node_systemd_unit_state{instance=%s,name=%s,state="%s"}' % (repr(instance), repr(name), state),
                "node %s is %s service %s" % (node.hostname, "not running" if should_run else "running", name),
                # in the case that the service has never ran on this host, the node exporter won't report it, so that's
                # fine -- as long as we didn't expect it to be running anyway.
                accept_missing=(not should_run))
    print("validated state of %d services" % len(servicemap["services"]))
    expect_prometheus_query_exact('sum(node_systemd_system_running)', len(config.nodes), "service management is running")


@command.wrap
def check_etcd_health():
    "verify that etcd is healthy and working"
    config = configuration.get_config()
    master_node_count = len([node for node in config.nodes if node.kind == "master"])
    expect_prometheus_query_exact('sum(etcd_server_has_leader)', master_node_count, "etcd servers are online")
    if float(pull_prometheus_query('sum(rate(etcd_server_proposals_committed_total[1m]))')) < 1:
        command.fail("etcd is not committing any proposals; is likely not healthy")
    print("all", master_node_count, "etcd servers seems to be healthy!")


@command.wrap
def check_kube_init():
    "verify that kubernetes appears initialized"
    config = configuration.get_config()
    master_node_count = len([node for node in config.nodes if node.kind == "master"])
    expect_prometheus_query_exact('sum(up{job="kubernetes-apiservers"})', master_node_count, "kubernetes apiservers are online")
    print("all", master_node_count, "kubernetes apiservers seem to be online!")


@command.wrap
def check_kube_health():
    "verify that kubernetes appears healthy"

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


@command.wrap
def check_pull():
    "verify that container pulling from the homeworld registry, and associated container execution, are functioning"

    config = configuration.get_config()
    node_count = len([node for node in config.nodes if node.kind != "supervisor"])
    expect_prometheus_query_exact("sum(oci_pull_check)", node_count, "nodes are pulling ocis properly")
    expect_prometheus_query_exact("sum(oci_exec_check)", node_count, "nodes are launching ocis properly")
    print("oci pulling seems to work!")


@command.wrap
def check_flannel():
    "verify that the flannel addon is functioning"

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


@command.wrap
def check_dns():
    "verify that the DNS addon is functioning"

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


def get_service_ip(service_name: str) -> str:
    clusterIP = access.call_kubectl(["get", "service", "-o=jsonpath={.spec.clusterIP}", "--", service_name],
                                    return_result=True).decode().strip()
    if clusterIP.count(".") != 3 or not clusterIP.replace(".", "").isdigit():
        command.fail("invalid clusterIP for %s service: %s" % (service_name, repr(clusterIP)))
    return clusterIP


# only verifiable if the local user grant certificates exist, which means that we generated them ourselves via the
# 'spire authority genupstream' command.
def is_user_grant_verifiable():
    user_key, user_cert = authority.get_local_grant_user_paths()
    return os.path.exists(user_key) and os.path.exists(user_cert)


@command.wrap
def check_user_grant():
    "verify that user-grant and its kubeconfigs work"
    config = configuration.get_config()

    # because we don't yet have load balancing, we have to somehow get *inside the cluster* to test this.
    # that means figuring out the IP address for the user-grant service, uploading the local user cert to the master
    # node, and then authenticating to user-grant via curl on the master node. bluh.
    # TODO: once load balancing is ready, make this whole thing much simpler

    # we use a master node so we're confident we aren't connecting to the node where user-grant is hosted. there's
    # nothing about this that otherwise requires it; usually we'd choose a worker node to avoid running unnecessary code
    # on the master nodes, but this is entirely for testing in non-production clusters, so it doesn't matter.
    proxy_node = config.get_any_node("master")

    service_ip = get_service_ip("user-grant")
    user_key, user_cert = authority.get_local_grant_user_paths()
    remote_key, remote_cert = "/etc/homeworld/testing/usergrant.key", "/etc/homeworld/testing/usergrant.pem"
    ssh.check_ssh(proxy_node, "rm", "-f", remote_key, remote_cert)
    ssh.check_ssh(proxy_node, "mkdir", "-p", "/etc/homeworld/testing")
    ssh.check_scp_up(proxy_node, user_key, remote_key)
    ssh.check_scp_up(proxy_node, user_cert, remote_cert)

    setup.modify_temporary_dns(proxy_node, {config.user_grant_domain: service_ip})
    try:
        kubeconfig = ssh.check_ssh_output(proxy_node, "curl", "--key", remote_key, "--cert", remote_cert,
                                          "https://%s/" % config.user_grant_domain).decode()
    finally:
        setup.modify_temporary_dns(proxy_node, {})

    magic_phrase = "it allows authenticating to the Hyades cluster as you"
    if magic_phrase not in kubeconfig:
        command.fail("invalid kubeconfig: did not see phrase " + repr(magic_phrase),
                     "kubeconfig received read as follows: " + repr(kubeconfig))

    print("successfully retrieved kubeconfig from user-grant!")

    # at this point, we have a kubeconfig generated by user-grant, and now we want to confirm that it works.
    # we'll confirm that the kubeconfig works by checking that the auto-created rolebinding passes the sniff test.

    with tempfile.TemporaryDirectory() as workdir:
        kubeconfig_path = os.path.join(workdir, "granted-kubeconfig")
        util.writefile(kubeconfig_path, kubeconfig.encode())

        rolebinding = json.loads(
            subprocess.check_output(["hyperkube", "kubectl", "--kubeconfig", kubeconfig_path, "-o", "json",
                                     "get", "rolebindings", "auto-grant-" + authority.UPSTREAM_USER_NAME]).decode())

        if rolebinding.get("roleRef", {}).get("name") != "admin":
            command.fail("rolebinding for user was not admin in %s" % repr(rolebinding))

    print("autogenerated rolebinding for user", repr(authority.UPSTREAM_USER_NAME), "passed basic check!")


main_command = command.Mux("commands about verifying the state of a cluster", {
    "keystatics": check_keystatics,
    "keygateway": check_keygateway,
    "supervisor-accessible": check_supervisor_accessible,
    "online": check_online,
    "ssh-with-certs": check_ssh_with_certs,
    "supervisor-certs": check_certs_on_supervisor,
    "systemd": check_systemd_services,
    "etcd": check_etcd_health,
    "kubernetes-init": check_kube_init,
    "kubernetes": check_kube_health,
    "pull": check_pull,
    "flannel": check_flannel,
    "dns-addon": check_dns,
    "user-grant": check_user_grant,
})
