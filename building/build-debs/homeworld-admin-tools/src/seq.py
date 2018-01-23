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
    ops.add_operation("verify ssh access to supervisor", iterative_verifier(verify.check_ssh_with_certs, 20.0))

    ops.print_annotations("set up ssh")


def sequence_supervisor(ops: setup.Operations) -> None:
    ops.add_subcommand(sequence_keysystem)
    ops.add_subcommand(setup.setup_prometheus)
    ops.add_subcommand(sequence_ssh)
    ops.add_subcommand(setup.setup_bootstrap_registry)

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


def sequence_cluster(ops: setup.Operations) -> None:
    ops.add_operation("verify that the fundamental cluster infrastructure is online",
                      iterative_verifier(verify.check_online, 120.0))

    ops.add_subcommand(setup.setup_dns_bootstrap)
    ops.add_subcommand(setup.setup_services)

    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 120.0))
    ops.add_operation("verify that kubernetes has partially configured successfully",
                      iterative_verifier(verify.check_kube_init, 120.0))

    ops.add_operation("deploy or update flannel", deploy.launch_flannel)
    ops.add_operation("deploy or update kube-state-metrics", deploy.launch_kube_state_metrics)
    ops.add_operation("deploy or update dns-addon", deploy.launch_dns_addon)
    ops.add_operation("deploy or update flannel-monitor", deploy.launch_flannel_monitor)
    ops.add_operation("deploy or update dns-monitor", deploy.launch_dns_monitor)

    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 180.0))
    ops.add_operation("verify that acis can be pulled from the registry", verify.check_aci_pull)
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel, 120.0))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns, 120.0))

    ops.print_annotations("set up the kubernetes cluster")


main_command = command.mux_map("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": setup.wrapop("set up and verify functionality of the keyserver and keygateway", sequence_keysystem),
    "ssh": setup.wrapop("set up and verify ssh access to the supervisor node", sequence_ssh),
    "supervisor": setup.wrapop("set up and verify functionality of entire supervisor node (keysystem + ssh)",
                               sequence_supervisor),
    "cluster": setup.wrapop("set up and verify kubernetes infrastructure operation", sequence_cluster),
})
