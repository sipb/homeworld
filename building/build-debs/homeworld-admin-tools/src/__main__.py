import sys

import command
import config
import authority
import iso

main_command = command.mux_map("invoke a top-level command", {
    "iso": iso.main_command,
    "config": config.main_command,
    "authority": authority.main_command,
})


if __name__ == "__main__":
    sys.exit(command.main_invoke(main_command, sys.argv[1:]))
