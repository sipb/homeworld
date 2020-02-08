import command
import resource


def get_git_version():
    return resource.get("//version:GIT_VERSION").decode().rstrip()


def get_apt_branch():
    return resource.get("//upload:BRANCH_NAME").decode().rstrip()


def get_apt_url():
    return resource.get("//upload:DOWNLOAD_URL").decode().rstrip()


@command.wrap
def display_version():
    "display version info"
    git_version = get_git_version()
    print("Git commit hash:", git_version)

    apt_branch = get_apt_branch()
    print("Apt branch:", apt_branch)

    apt_url = get_apt_url()
    print("Apt URL:", apt_url)


version_command = display_version
