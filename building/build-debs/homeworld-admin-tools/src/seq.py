import time

import access
import command
import deploy
import setup
import verify


def sequence_keysystem(ops: setup.Operations) -> None:
    ops.add_subcommand(setup.setup_keyserver)
    ops.add_operation("verify that keyserver static files can be fetched",
        iterative_verifier(verify.check_keystatics, 10.0))
    ops.add_subcommand(setup.admit_keyserver)
    ops.add_subcommand(setup.setup_keygateway)
    ops.add_operation("verify that the keygateway is responsive", verify.check_keygateway)

    ops.print_annotations("set up the keysystem")


def sequence_ssh(ops: setup.Operations) -> None:
    ops.add_operation("request SSH access to cluster", access.access_ssh_with_add)
    ops.add_subcommand(setup.setup_supervisor_ssh)
    ops.add_operation("verify ssh access to supervisor", verify.check_ssh_with_certs)

    ops.print_annotations("set up ssh")


def sequence_supervisor(ops: setup.Operations) -> None:
    ops.add_subcommand(sequence_keysystem)
    ops.add_subcommand(setup.setup_prometheus)
    ops.add_subcommand(sequence_ssh)

    ops.print_annotations("set up the keysystem")


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

    ver.dispatch_set_name = lambda name: command.provide_command_for_function(verifier, name)
    ver.dispatch_get_name = lambda default: command.get_command_for_function(verifier, default)

    return ver


def sequence_core(ops: setup.Operations) -> None:
    ops.add_subcommand(setup.setup_services)

    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 20.0))

    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 10.0))

    ops.print_annotations("set up the core kubernetes cluster")


def sequence_registry(ops: setup.Operations) -> None:
    ops.add_subcommand(setup.setup_dns_bootstrap)
    ops.add_subcommand(setup.setup_bootstrap_registry)
    ops.add_operation("verify that acis can be pulled from the registry", verify.check_aci_pull)

    ops.print_annotations("set up the bootstrap container registry")


def sequence_flannel(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update flannel", deploy.launch_flannel)
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel_kubeinfo, 60.0))
    ops.add_operation("verify that flannel is functioning", verify.check_flannel_function)

    ops.print_annotations("set up flannel")


def sequence_dns_addon(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update dns-addon", deploy.launch_dns_addon)
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns_kubeinfo, 60.0))
    ops.add_operation("verify that dns-addon is functioning", verify.check_dns_function)

    ops.print_annotations("set up the dns-addon")


def sequence_addons(ops: setup.Operations) -> None:
    ops.add_operation("deploy or update flannel", deploy.launch_flannel)
    ops.add_operation("deploy or update dns-addon", deploy.launch_dns_addon)

    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel_kubeinfo, 60.0))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns_kubeinfo, 60.0))

    ops.add_operation("verify that flannel is functioning", verify.check_flannel_function)
    ops.add_operation("verify that dns-addon is functioning", verify.check_dns_function)

    ops.print_annotations("set up the dns-addon")


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
