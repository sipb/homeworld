import sys

import command
import configuration
import authority
import iso
import setup
import query
import verify
import access
import infra
import keys
import seq
import deploy
import metadata
import virt

main_command = command.Mux("invoke a top-level command", {
    "iso": iso.main_command,
    "config": configuration.main_command,
    "authority": authority.main_command,
    "keytab": keys.keytab_command,
    "https": keys.https_command,
    "setup": setup.main_command,
    "query": query.main_command,
    "verify": verify.main_command,
    "access": access.main_command,
    "etcdctl": access.etcdctl_command,
    "kubectl": access.kubectl_command,
    "foreach": access.foreach_command,
    "infra": infra.main_command,
    "seq": seq.main_command,
    "deploy": deploy.main_command,
    "virt": virt.main_command,
    "version": metadata.version_command,
})


if __name__ == "__main__":
    sys.exit(command.main_invoke(main_command))
