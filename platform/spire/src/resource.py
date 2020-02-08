import pkgutil

import util


def get(path: str) -> bytes:
    if not path.startswith("//") or ":" not in path:
        raise ValueError("expected path %s passed to resource.get to be of the form //<package>:<filename>" % repr(path))
    package, filename = path[2:].split(":", 1)
    result = pkgutil.get_data(package.replace("/", "."), filename)
    if result is None:
        raise Exception("no such package found: %s" % package)
    return result


def extract(path: str, fileout: str) -> None:
    util.writefile(fileout, get(path))
