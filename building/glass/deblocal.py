import apt_pkg
import functools
import os

import project

apt_pkg.init_system()

version_key = functools.cmp_to_key(apt_pkg.version_compare)


# TODO: consider grabbing version from glass.yaml of each component, rather than "latest built binary"?
def resolve_package(name: str, branch: str):
    base = project.get_bindir(branch)
    choices = {}
    for binary in os.listdir(base):
        if binary.count("_") != 2:
            continue
        package, version, suffix = binary.split("_")
        if package == name and suffix == "amd64.deb":
            choices[version] = os.path.join(base, binary)
    if not choices:
        raise Exception("cannot find compiled package %s on branch %s" % (name, branch))
    best_version = max(choices.keys(), key=version_key)
    return choices[best_version]
