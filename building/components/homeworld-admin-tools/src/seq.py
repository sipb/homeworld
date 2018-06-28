import time
import argparse

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


def sequence_ssh(ops: setup.Operations) -> None:
    ops.add_operation("request SSH access to cluster", access.access_ssh_with_add)
    ops.add_subcommand(setup.revoke_direct_ssh)
    ops.add_operation("verify ssh access to supervisor", iterative_verifier(verify.check_ssh_with_certs, 20.0))


def sequence_supervisor(ops: setup.Operations) -> None:
    ops.add_subcommand(sequence_keysystem)
    ops.add_operation("verify that kubernetes certs are available on supervisor", iterative_verifier(verify.check_certs_on_supervisor, 20.0))
    ops.add_subcommand(setup.setup_prometheus)
    ops.add_subcommand(sequence_ssh)
    ops.add_subcommand(setup.setup_bootstrap_registry)

    ops.add_operation("pre-deploy flannel", deploy.launch_flannel)
    ops.add_operation("pre-deploy dns-addon", deploy.launch_dns_addon)
    ops.add_operation("pre-deploy flannel-monitor", deploy.launch_flannel_monitor)
    ops.add_operation("pre-deploy dns-monitor", deploy.launch_dns_monitor)


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

    command.provide_command_for_function(ver, command.get_command_for_function(verifier))
    ver.dispatch_get_name = lambda default: command.get_command_for_function(verifier)

    return ver


def sequence_cluster(ops: setup.Operations) -> None:
    ops.add_operation("verify that the fundamental cluster infrastructure is online",
                      iterative_verifier(verify.check_online, 120.0))

    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 120.0))
    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 120.0))

    ops.add_operation("verify that acis can be pulled from the registry", iterative_verifier(verify.check_aci_pull, 60.0))
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel, 180.0))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns, 120.0))


def add_dry_run_argument(parser: argparse.ArgumentParser, dest: str):
    parser.add_argument("--dry-run", dest=dest, action="store_true", help="show command sequence performed by command without actually running them")


def wrapseq(desc: str, f):
    def wrap_param_tx(args):
        ops = setup.Operations()

        def invoke():
            if args.dry_run or args.dry_run_outer:
                ops.print_annotations()
            else:
                ops.run_operations()

        return [ops] + args.params, invoke

    desc, inner_configure = command.wrap(desc, f, wrap_param_tx)

    def configure(command: list, parser: argparse.ArgumentParser):
        add_dry_run_argument(parser, "dry_run")
        inner_configure(command, parser)

    return desc, configure


def seq_mux_map(desc, mapping):
    desc, inner_configure = command.mux_map(desc, mapping)

    def configure(command: list, parser: argparse.ArgumentParser):
        # allow --dry-run to be present before selector and also have it appear in the help message
        add_dry_run_argument(parser, "dry_run_outer")
        inner_configure(command, parser)

    return desc, configure


main_command = seq_mux_map("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": wrapseq("set up and verify functionality of the keyserver and keygateway", sequence_keysystem),
    "ssh": wrapseq("set up and verify ssh access to the supervisor node", sequence_ssh),
    "supervisor": wrapseq("set up and verify functionality of entire supervisor node (keysystem + ssh)",
                               sequence_supervisor),
    "cluster": wrapseq("set up and verify kubernetes infrastructure operation", sequence_cluster),
})
