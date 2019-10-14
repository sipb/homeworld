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


def launch_spec(spec_name, extra_kvs: dict=None, export=False):
    config = configuration.get_config()
    spec = configuration.get_single_kube_spec(spec_name, extra_kvs).encode()
    if export:
        util.writefile(spec_name, spec)
    else:
        for node in config.nodes:
            if node.kind == "supervisor":
                ssh.check_ssh(node, "mkdir", "-p", DEPLOYQUEUE)
                ssh.upload_bytes(node, spec, "%s/%d.%s" % (DEPLOYQUEUE, int(time.time()), spec_name))
                print("Uploaded spec to deployqueue.")


def launch_spec_direct(spec_name): # TODO: add a flag that enables this instead of launch_spec
    with tempfile.TemporaryDirectory() as d:
        specfile = os.path.join(d, "spec.yaml")
        util.writefile(specfile, configuration.get_single_kube_spec(spec_name).encode())
        access.call_kubectl(["apply", "-f", specfile], return_result=False)


def launch_flannel(export: bool=False):
    launch_spec("flannel.yaml", export=export)


def launch_flannel_monitor(export: bool=False):
    launch_spec("flannel-monitor.yaml", export=export)


def launch_dns_addon(export: bool=False):
    launch_spec("dns-addon.yaml", export=export)


def launch_dns_monitor(export: bool=False):
    launch_spec("dns-monitor.yaml", export=export)


def launch_user_grant(export: bool=False):
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

    upstream_cert_path = os.path.join(configuration.get_project(), "user-grant-upstream.pem")
    if not os.path.exists(upstream_cert_path):
        command.fail("user-grant-upstream.pem not found in homeworld directory")
    upstream_cert = util.readfile(upstream_cert_path).decode()
    launch_spec("user-grant.yaml", {
        "SERVER_KEY_BASE64": skey64.decode(),
        "SERVER_CERT_BASE64": scert64.decode(),
        "ISSUER_KEY_BASE64": ikey64.decode(),
        "ISSUER_CERT_BASE64": icert64.decode(),
        "EMAIL_DOMAIN": config.user_grant_email_domain,
        "UPSTREAM_CERTIFICATE": upstream_cert,
    }, export=export)


main_command = command.mux_map("commands to deploy systems onto the kubernetes cluster", {
    "flannel": command.wrap("deploy the specifications to run flannel", launch_flannel),
    "flannel-monitor": command.wrap("deploy the specifications to run the flannel monitor", launch_flannel_monitor),
    "dns-addon": command.wrap("deploy the specifications to run the dns-addon", launch_dns_addon),
    "dns-monitor": command.wrap("deploy the specifications to run the dns monitor", launch_dns_monitor),
    "user-grant": command.wrap("deploy the specifications to run the user grant website", launch_user_grant),
})
