import sys
import hashlib

# NOTE: this hash is not intended to have any security properties. it is only used for caching.

args = sys.argv[1:]
files = args[:args.index("--")]


def sha256(s):
    if type(s) != bytes:
        s = s.encode()
    return hashlib.sha256(s).digest()


def sha256_file(f):
    h = hashlib.sha256()
    while True:
        buf = f.read(4096)
        if not buf:
            return h.digest()
        h.update(buf)


hashes = [sha256(b"".join(sha256(arg) for arg in args))]
for input in files:
    if input == "--empty":
        hashes.append(b"(empty)")  # not a real hash, but we're going to hash it again, so it's okay
    else:
        with open(input, "rb") as f:
            hashes.append(sha256_file(f))

print(hashlib.sha256(b"".join(hashes)).hexdigest())
