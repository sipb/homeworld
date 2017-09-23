import sys

import command
import iso

main_command = command.mux_map("invoke a top-level command", {
    "iso": iso.main_command,
})


if __name__ == "__main__":
    sys.exit(command.main_invoke(main_command, sys.argv[1:]))
