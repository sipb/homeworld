import base64
import os
import tempfile
import time

import access
import authority
import command
import configuration
import keys
import ssh
import util

DEPLOYQUEUE = "/etc/homeworld/deployqueue"


def launch_spec(path, extra_kvs: dict=None, export=False):
    config = configuration.get_config()
    spec = configuration.get_single_kube_spec(path, extra_kvs).encode()
    assert path[:2] == '//'
    yamlname = path[2:].replace(":", "-")
    if export:
        util.writefile(yamlname, spec)
    else:
        for node in config.nodes:
            if node.kind == "supervisor":
                ssh.check_ssh(node, "mkdir", "-p", DEPLOYQUEUE)
                ssh.upload_bytes(node, spec, "%s/%f.%s" % (DEPLOYQUEUE, time.time(), yamlname))
                print("Uploaded spec to deployqueue.")


def launch_spec_direct(path): # TODO: add a flag that enables this instead of launch_spec
    with tempfile.TemporaryDirectory() as d:
        specfile = os.path.join(d, "spec.yaml")
        util.writefile(specfile, configuration.get_single_kube_spec(path).encode())
        access.call_kubectl(["apply", "-f", specfile], return_result=False)


@command.wrap
def launch_flannel(export: bool=False):
    "deploy the specifications to run flannel"
    launch_spec("//flannel:kubernetes.yaml", export=export)


@command.wrap
def launch_flannel_monitor(export: bool=False):
    "deploy the specifications to run the flannel monitor"
    launch_spec("//flannel-monitor:kubernetes.yaml", export=export)


@command.wrap
def launch_dns_addon(export: bool=False):
    "deploy the specifications to run the dns-addon"
    launch_spec("//dnsmasq:kubernetes.yaml", export=export)


@command.wrap
def launch_dns_monitor(export: bool=False):
    "deploy the specifications to run the dns monitor"
    launch_spec("//dns-monitor:kubernetes.yaml", export=export)


@command.wrap
def launch_user_grant(export: bool=False):
    "deploy the specifications to run the user grant website"
    config = configuration.get_config()
    if config.user_grant_domain == '':
        command.fail("no user_grant_domain specified in setup.yaml")
    if config.user_grant_email_domain == '':
        command.fail("no user_grant_email_domain specified in setup.yaml")
    skey, scert = keys.decrypt_https(config.user_grant_domain)
    skey64, scert64 = base64.b64encode(skey), base64.b64encode(scert)

    ikey = authority.get_decrypted_by_filename("./kubernetes.key")
    icert = authority.get_pubkey_by_filename("./kubernetes.pem")
    ikey64, icert64 = base64.b64encode(ikey), base64.b64encode(icert)

    _, upstream_cert_path = authority.get_upstream_cert_paths()
    if not os.path.exists(upstream_cert_path):
        command.fail("user-grant-upstream.pem not found in homeworld directory")
    upstream_cert = util.readfile(upstream_cert_path).decode()
    launch_spec("//user-grant:kubernetes.yaml", {
        "SERVER_KEY_BASE64": skey64.decode(),
        "SERVER_CERT_BASE64": scert64.decode(),
        "ISSUER_KEY_BASE64": ikey64.decode(),
        "ISSUER_CERT_BASE64": icert64.decode(),
        "EMAIL_DOMAIN": config.user_grant_email_domain,
        "UPSTREAM_CERTIFICATE": upstream_cert,
    }, export=export)


main_command = command.Mux("commands to deploy systems onto the kubernetes cluster", {
    "flannel": launch_flannel,
    "flannel-monitor": launch_flannel_monitor,
    "dns-addon": launch_dns_addon,
    "dns-monitor": launch_dns_monitor,
    "user-grant": launch_user_grant,
})
