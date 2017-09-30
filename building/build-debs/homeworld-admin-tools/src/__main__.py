import sys

import command
import configuration
import authority
import iso
import setup

main_command = command.mux_map("invoke a top-level command", {
    "iso": iso.main_command,
    "config": configuration.main_command,
    "authority": authority.main_command,
    "setup": setup.main_command,
})


if __name__ == "__main__":
    sys.exit(command.main_invoke(main_command, sys.argv[1:]))
