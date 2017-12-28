import os
import command
from resources import get_resource


def display_version():
    deb_version = get_resource("DEB_VERSION").decode().rstrip()
    print("Debian package version:", deb_version)

    git_version = get_resource("GIT_VERSION").decode().rstrip()
    print("Git commit hash:", git_version)


main_command = command.wrap("display version info", display_version)
