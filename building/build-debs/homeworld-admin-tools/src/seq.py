import command
import time
import deploy
import infra
import access
import configuration
import setup
import verify


def sequence_keysystem(ops: setup.Operations) -> None:
    # ** Configure the supervisor keyserver **

    # spire setup keyserver
    setup.setup_keyserver(ops)
    # spire verify keystatics
    ops.add_operation("verify that keyserver static files can be fetched", verify.check_keystatics)

    # ** Admit the supervisor node to the cluster **

    # spire setup self-admit
    setup.admit_keyserver(ops)

    # ** Prepare kerberos gateway **

    # spire setup keygateway
    setup.setup_keygateway(ops)
    # spire verify keygateway
    ops.add_operation("verify that the keygateway is responsive", verify.check_keygateway)


def sequence_ssh(ops: setup.Operations) -> None:
    # ** Request SSH cert **

    # spire access ssh
    # (if this fails, you might need to make sure you don't have any stale kerberos tickets)
    ops.add_operation("request SSH access to cluster", access.access_ssh_with_add)

    # ** Configure and test SSH **

    # spire setup supervisor-ssh
    setup.setup_supervisor_ssh(ops)
    # spire verify ssh-with-certs
    ops.add_operation("verify ssh access to supervisor", verify.check_ssh_with_certs)


def sequence_supervisor(ops: setup.Operations) -> None:
    # spire seq keysystem
    sequence_keysystem(ops)
    # spire seq ssh
    sequence_ssh(ops)


def iterative_verifier(verifier, max_time, pause=2.0):
    def ver():
        end_time = time.time() + max_time
        while True:
            try:
                verifier()
                return
            except Exception as e:
                if time.time() >= end_time:
                    raise e
                print("Verification failed:", e)
                print("RETRYING...")
            time.sleep(pause)

    return ver


def sequence_core(ops: setup.Operations) -> None:
    # ** Install and upgrade packages on all systems **
    # spire infra install-packages
    infra.infra_install_packages(ops)

    # ** Launch services **
    # spire setup services
    setup.setup_services(ops)

    # spire verify etcd
    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 20.0))
    # spire verify kubernetes
    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 10.0))


def sequence_registry(ops: setup.Operations) -> None:
    setup.setup_dns_bootstrap(ops)
    setup.setup_bootstrap_registry(ops)
    ops.add_operation("verify that acis can be pulled from the registry", verify.check_aci_pull)


def sequence_flannel(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update flannel", lambda: deploy.launch_spec("flannel.yaml"))
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel_kubeinfo, 60.0))
    ops.add_operation("verify that flannel is functioning", verify.check_flannel_function)


def sequence_dns_addon(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update dns-addon", lambda: deploy.launch_spec("dns-addon.yaml"))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns_kubeinfo, 60.0))
    ops.add_operation("verify that dns-addon is functioning", verify.check_dns_function)


def sequence_addons(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update flannel", lambda: deploy.launch_spec("flannel.yaml"))
    ops.add_operation("deploy or update dns-addon", lambda: deploy.launch_spec("dns-addon.yaml"))
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel_kubeinfo, 60.0))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns_kubeinfo, 10.0))
    ops.add_operation("verify that flannel is functioning", verify.check_flannel_function)
    ops.add_operation("verify that dns-addon is functioning", verify.check_dns_function)


main_command = command.mux_map("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": setup.wrapop("set up and verify functionality of the keyserver and keygateway", sequence_keysystem),
    "ssh": setup.wrapop("set up and verify ssh access to the supervisor node", sequence_ssh),
    "supervisor": setup.wrapop("set up and verify functionality of entire supervisor node (keysystem + ssh)", sequence_supervisor),
    "core": setup.wrapop("set up and verify core infrastructure operation", sequence_core),
    "registry": setup.wrapop("set up and verify the bootstrap container registry", sequence_registry),
    "flannel": setup.wrapop("set up and verify the flannel core service", sequence_flannel),
    "dns-addon": setup.wrapop("set up and verify the dns-addon core service", sequence_dns_addon),
    "addons": setup.wrapop("set up and verify the flannel and dns-addon core services", sequence_addons),
})
