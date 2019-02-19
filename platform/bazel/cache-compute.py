import json
import sys


def load(path):
    with open(path, "r") as f:
        return f.read().strip()


args = sys.argv[1:]
assert len(args) % 3 == 0, "must be a multiple of three arguments for (name, hash, version) triplets, not: %d" % len(sys.argv[1:])

version_cache = {}

for name_file, hash_file, version_file in zip(args[0::3], args[1::3], args[2::3]):
    name = load(name_file)
    assert name not in version_cache, "collision between two packages with the name: %s" % name
    version_cache[name] = {
        "hash": load(hash_file),
        "version": load(version_file),
    }

print(json.dumps(version_cache))
