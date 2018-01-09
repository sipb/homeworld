import os
import command
from resources import get_resource


def get_git_version():
    return get_resource("GIT_VERSION").decode().rstrip()


def display_version():
    deb_version = get_resource("DEB_VERSION").decode().rstrip()
    print("Debian package version:", deb_version)

    git_version = get_git_version()
    print("Git commit hash:", git_version)


main_command = command.wrap("display version info", display_version)
