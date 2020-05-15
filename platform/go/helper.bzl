def importpath_to_repo_name(importpath):
    """
    Takes in a path like "github.com/coreos/go-systemd" and produces a
    repository name like "com_github_coreos_go_systemd".
    """
    parts = importpath.lower().replace("-", "_").split("/")
    hostname_parts = parts[0].split(".")
    return "_".join(hostname_parts[::-1] + parts[1:]).replace(".", "_")
