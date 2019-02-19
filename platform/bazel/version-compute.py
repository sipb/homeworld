import sys
import json

production_version_path, package_name, hashfile_path = sys.argv[1:]

with open(production_version_path, "r") as f:
    production_version = f.read().strip()

with open(hashfile_path, "r") as f:
    hash = f.read().strip()

assert production_version.replace(".", "").isdigit() and production_version.count(".") == 2 and ".." not in "." + production_version + "."
assert hash.isalnum() and len(hash) == 64, "bad hash: %s" % hash

# for now, just use the production version, regardless of the value of the hash.

print(production_version)
