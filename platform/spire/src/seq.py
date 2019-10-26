import time
import argparse

import access
import command
import deploy
import infra
import setup
import configuration
import verify


def sequence_keysystem(ops: setup.Operations) -> None:
    ops.add_subcommand(setup.setup_keyserver)
    ops.add_operation("verify that keyserver static files can be fetched",
                      iterative_verifier(verify.check_keystatics, 60.0))
    ops.add_subcommand(setup.admit_keyserver)
    if configuration.get_config().is_kerberos_enabled():
        ops.add_subcommand(setup.setup_keygateway)
        ops.add_operation("verify that the keygateway is responsive", verify.check_keygateway)
    else:
        ops.add_operation("skip keygateway enablement (kerberos is disabled)", lambda: None)


def sequence_ssh(ops: setup.Operations) -> None:
    ops.add_operation("request SSH access to cluster", access.access_ssh_with_add)
    ops.add_subcommand(setup.setup_supervisor_ssh)
    ops.add_operation("verify ssh access to supervisor", iterative_verifier(verify.check_ssh_with_certs, 20.0))


def sequence_supervisor(ops: setup.Operations) -> None:
    config = configuration.get_config()
    ops.add_subcommand(sequence_keysystem)
    ops.add_operation("verify that keysystem certs are available on supervisor", iterative_verifier(verify.check_certs_on_supervisor, 20.0))
    ops.add_subcommand(setup.setup_prometheus)
    ops.add_subcommand(sequence_ssh)
    ops.add_subcommand(setup.setup_bootstrap_registry)
    ops.add_subcommand(setup.update_registry)

    ops.add_operation("pre-deploy flannel", deploy.launch_flannel)
    ops.add_operation("pre-deploy dns-addon", deploy.launch_dns_addon)
    ops.add_operation("pre-deploy flannel-monitor", deploy.launch_flannel_monitor)
    ops.add_operation("pre-deploy dns-monitor", deploy.launch_dns_monitor)

    if config.user_grant_domain != '':
        ops.add_operation("pre-deploy user-grant", deploy.launch_user_grant)
    else:
        ops.add_operation("skip pre-deploying user-grant (not configured)", lambda: None)

    # TODO: have a way to do this without a specialized just-for-supervisor method
    ops.add_subcommand(infra.infra_sync_supervisor)


class IterativeVerifier(command.Simple):
    def __init__(self, verifier, max_time, pause=2.0):
        super().__init__(verifier)
        self.verifier = verifier
        self.max_time = max_time
        self.pause = pause
        self.func = self._verify_loop

    def _verify_loop(self, *args, **kwargs):
        end_time = time.time() + self.max_time
        while True:
            try:
                self.verifier(*args, **kwargs)
                return
            except Exception as e:
                if time.time() >= end_time:
                    print("Timeout - no more retries.")
                    raise e
                print("Verification failed:", e)
                print("RETRYING...")
            time.sleep(self.pause)

    def command(self, *args, **kwargs):
        if self._command is None and isinstance(self.verifier, command.Command):
            return self.verifier.command(*args, **kwargs)
        return super().command(*args, **kwargs)

def iterative_verifier(f, *args, **kwargs):
    return functools.update_wrapper(IterativeVerifier(f, *args, **kwargs), f, updated=[])


def sequence_cluster(ops: setup.Operations) -> None:
    ops.add_operation("verify that the fundamental cluster infrastructure is online",
                      iterative_verifier(verify.check_online, 120.0))

    ops.add_operation("verify that etcd has launched successfully",
                      iterative_verifier(verify.check_etcd_health, 120.0))
    ops.add_operation("verify that kubernetes has launched successfully",
                      iterative_verifier(verify.check_kube_health, 120.0))

    ops.add_operation("verify that containers can be pulled from the registry", iterative_verifier(verify.check_pull, 120.0))
    ops.add_operation("verify that flannel is online", iterative_verifier(verify.check_flannel, 210.0))
    ops.add_operation("verify that dns-addon is online", iterative_verifier(verify.check_dns, 120.0))

    if verify.is_user_grant_verifiable():
        ops.add_operation("verify that user-grant is working properly", iterative_verifier(verify.check_user_grant, 120.0))
    elif configuration.get_config().user_grant_domain != '':
        ops.add_operation("skip verifying user-grant (no client certificate)", lambda: None)
    else:
        ops.add_operation("skip verifying user-grant (not configured)", lambda: None)


main_command = seq_mux_map("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": wrapseq("set up and verify functionality of the keyserver and keygateway", sequence_keysystem),
    "ssh": wrapseq("set up and verify ssh access to the supervisor node", sequence_ssh),
    "supervisor": wrapseq("set up and verify functionality of entire supervisor node (keysystem + ssh)",
                               sequence_supervisor),
    "cluster": wrapseq("set up and verify kubernetes infrastructure operation", sequence_cluster),
})
