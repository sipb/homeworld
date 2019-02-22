import sys
import json

production_version_path, version_cache_path, package_name, hashfile_path = sys.argv[1:]

with open(production_version_path, "r") as f:
    production_version = f.read().strip()

with open(version_cache_path, "r") as f:
    version_cache = json.load(f)

with open(hashfile_path, "r") as f:
    hash = f.read().strip()

assert production_version.replace(".", "").isdigit() and production_version.count(".") == 2 and ".." not in "." + production_version + "."
assert hash.isalnum() and len(hash) == 64, "bad hash: %s" % hash


def is_later(a, b):
    if not a:
        return False
    elif not b:
        return True
    elif a[0] == b[0]:
        return is_later(a[1:], b[1:])
    elif a[0] > b[0]:
        return True
    else:
        return False


def latest_version(a, b):
    ai = [int(v) for v in a.split(".")]
    bi = [int(v) for v in b.split(".")]
    if is_later(ai, bi):
        return a
    else:
        return b


def increment_version(x):
    parts = x.split(".")
    assert parts[-1] == str(int(parts[-1]))
    parts[-1] = str(int(parts[-1]) + 1)
    return ".".join(parts)


if package_name not in version_cache:
    version = production_version
else:
    info = version_cache[package_name]
    version = info["version"]
    if info["hash"] != hash:
        version = increment_version(version)
    version = latest_version(version, production_version)

print(version)
