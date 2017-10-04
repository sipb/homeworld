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

main_command = command.mux_map("invoke a top-level command", {
    "iso": iso.main_command,
    "config": configuration.main_command,
    "authority": authority.main_command,
    "setup": setup.main_command,
    "query": query.main_command,
    "verify": verify.main_command,
    "access": access.main_command,
    "infra": infra.main_command,
})


if __name__ == "__main__":
    sys.exit(command.main_invoke(main_command, sys.argv[1:]))
