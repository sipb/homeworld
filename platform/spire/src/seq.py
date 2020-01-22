import functools
import time

import access
import command
import configuration
import deploy
import infra
import setup
import verify


@command.wrapseq
def sequence_keysystem(ops: command.Operations, skip_verify_keygateway: bool=False) -> None:
    "set up and verify functionality of the keyserver and keygateway"
    ops.add_command(iterative_verifier(verify.check_supervisor_accessible, 30.0))
    ops.add_subcommand(setup.setup_keyserver)
    ops.add_command(iterative_verifier(verify.check_keystatics, 60.0))
    ops.add_subcommand(setup.admit_keyserver)
    if configuration.get_config().is_kerberos_enabled():
        ops.add_subcommand(setup.setup_keygateway)
        if not skip_verify_keygateway:
            ops.add_command(verify.check_keygateway)
        else:
            ops.add_operation("skip keygateway verification", lambda: None)
    else:
        ops.add_operation("skip keygateway enablement (kerberos is disabled)", lambda: None)


@command.wrapseq
def sequence_ssh(ops: command.Operations) -> None:
    "set up and verify ssh access to the supervisor node"
    ops.add_command(access.access_ssh)
    ops.add_subcommand(setup.setup_supervisor_ssh)
    ops.add_command(iterative_verifier(verify.check_ssh_with_certs, 20.0))


@command.wrapseq
def sequence_supervisor(ops: command.Operations, skip_verify_keygateway: bool=False) -> None:
    "set up and verify functionality of entire supervisor node (keysystem + ssh)"
    config = configuration.get_config()
    ops.add_subcommand(sequence_keysystem, skip_verify_keygateway=skip_verify_keygateway)
    ops.add_command(iterative_verifier(verify.check_certs_on_supervisor, 20.0))
    ops.add_subcommand(setup.setup_prometheus)
    ops.add_subcommand(sequence_ssh)
    ops.add_subcommand(setup.setup_bootstrap_registry)
    ops.add_subcommand(setup.update_registry)

    ops.add_command(deploy.launch_flannel)
    ops.add_command(deploy.launch_dns_addon)
    ops.add_command(deploy.launch_flannel_monitor)
    ops.add_command(deploy.launch_dns_monitor)

    if config.user_grant_domain != '':
        ops.add_command(deploy.launch_user_grant)
    else:
        ops.add_operation("skip pre-deploying user-grant (not configured)", lambda: None)

    for node in config.nodes:
        if node.kind == 'supervisor':
            ops.add_subcommand(infra.infra_sync, node.hostname)

@command.wrapseq
def sequence_redeploy_config(ops: command.Operations) -> None:
    "redeploy a cluster configuration to a running cluster"
    # push new config to the keyserver
    setup.redeploy_keyserver(ops)
    # push new config to each keyclient and restart
    setup.redeploy_keyclients(ops)


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


@command.wrapseq
def sequence_cluster(ops: command.Operations) -> None:
    "set up and verify kubernetes infrastructure operation"

    ops.add_command(iterative_verifier(verify.check_online, 120.0))

    ops.add_command(iterative_verifier(verify.check_systemd_services, 120.0))

    ops.add_command(iterative_verifier(verify.check_etcd_health, 120.0))
    ops.add_command(iterative_verifier(verify.check_kube_health, 120.0))

    ops.add_command(iterative_verifier(verify.check_pull, 120.0))
    ops.add_command(iterative_verifier(verify.check_flannel_pods, 210.0))
    ops.add_command(iterative_verifier(verify.check_exec, 120.0))
    ops.add_command(iterative_verifier(verify.check_flannel, 120.0))
    ops.add_command(iterative_verifier(verify.check_dns, 120.0))

    if configuration.get_config().user_grant_domain == '':
        ops.add_operation("skip verifying user-grant (not configured)", lambda: None)
    elif not verify.is_user_grant_verifiable():
        ops.add_operation("skip verifying user-grant (no client certificate)", lambda: None)
    else:
        ops.add_operation("verify that user-grant is working properly", iterative_verifier(verify.check_user_grant, 120.0))


main_command = command.SeqMux("commands about running large sequences of cluster bring-up automatically", {
    "keysystem": sequence_keysystem,
    "ssh": sequence_ssh,
    "supervisor": sequence_supervisor,
    "cluster": sequence_cluster,
    "redeploy": sequence_redeploy_config,
})
