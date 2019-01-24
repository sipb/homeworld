import sys

with open(sys.argv[1], "r") as f:
    template = f.read()

kvs = {}
for arg in sys.argv[2:]:
    if not arg: continue
    if "=" in arg and ("<" not in arg or arg.index("<") > arg.index("=")):
        k, v = arg.split("=", 1)
        assert "<" not in k
        kvs[k] = v
    elif "<" in arg:
        k, f = arg.split("<", 1)
        assert "=" not in k
        with open(f, "r") as fi:
            kvs[k] = fi.read().strip()
    else:
        raise Exception("expected arguments to be K=V or K<F")

print(template.format(**kvs))
