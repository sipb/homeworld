import command
import time
import deploy
import infra
import access
import configuration
import setup
import verify


def sequence_keysystem(ops: setup.Operations, config: configuration.Config) -> None:
    # ** Configure the supervisor keyserver **

    # spire setup keyserver
    setup.setup_keyserver(ops, config)
    # spire verify keystatics
    ops.add_operation("verify that keyserver static files can be fetched", verify.check_keystatics)

    # ** Admit the supervisor node to the cluster **

    # spire setup self-admit
    setup.admit_keyserver(ops, config)

    # ** Prepare kerberos gateway **

    # spire setup keygateway
    setup.setup_keygateway(ops, config)
    # spire verify keygateway
    ops.add_operation("verify that the keygateway is responsive", verify.check_keygateway)


def sequence_ssh(ops: setup.Operations, config: configuration.Config) -> None:
    # ** Request SSH cert **

    # spire access ssh
    # (if this fails, you might need to make sure you don't have any stale kerberos tickets)
    ops.add_operation("request SSH access to cluster", access.access_ssh_with_add)

    # ** Configure and test SSH **

    # spire setup supervisor-ssh
    setup.setup_supervisor_ssh(ops, config)
    # spire verify ssh-with-certs
    ops.add_operation("verify ssh access to supervisor", verify.check_ssh_with_certs)


def iterative_verifier(verifier, max_time):
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
            time.sleep(1)

    return ver


def sequence_core(ops: setup.Operations, config: configuration.Config) -> None:
    # ** Install and upgrade packages on all systems **
    # spire infra install-packages
    infra.infra_install_packages(ops, config)

    # ** Launch services **
    # spire setup services
    setup.setup_services(ops, config)

    # spire verify etcd
    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 10.0))
    # spire verify kubernetes
    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 5.0))


def sequence_registry(ops: setup.Operations, config: configuration.Config) -> None:
    setup.setup_dns_bootstrap(ops, config)
    setup.setup_bootstrap_registry(ops, config)
    ops.add_operation("verify that acis can be pulled from the registry", verify.check_aci_pull)


main_command = command.mux_map("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": setup.wrapop("set up and verify functionality of the keyserver and keygateway", sequence_keysystem),
    "ssh": setup.wrapop("set up and verify ssh access to the supervisor node", sequence_ssh),
    "core": setup.wrapop("set up and verify core infrastructure operation", sequence_core),
    "registry": setup.wrapop("set up and verify the bootstrap container registry", sequence_registry),
})
