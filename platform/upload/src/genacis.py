import sys
import json
import tarfile

output, *acis = sys.argv[1:]
assert acis
assert len(acis) % 2 == 0  # (aci, sig) pairs

uploads = {}


def manifest_to_name(manifest):
    labels = {ent["name"]: ent["value"] for ent in manifest["labels"]}
    result = "{name}-{version}-{os}-{arch}.aci".format(
        name=manifest["name"],
        version=labels["version"],
        os=labels["os"],
        arch=labels["arch"],
    )
    # TODO: is this the right way to check that the name is reasonable?
    assert result.startswith("homeworld.private/") and result.count("/") == 1
    return result


def strip_name(x):
    assert x.startswith("bazel-out/k8-fastbuild/")
    return x.split("/",2)[2]


for aci, sig in zip(acis[::2], acis[1::2]):
    with tarfile.open(aci) as tf:
        with tf.extractfile("./manifest") as manifest:
            assert manifest is not None
            aciname = manifest_to_name(json.loads(manifest.read().decode()))
    uploads["aci/" + aciname] = "file:" + strip_name(aci)
    uploads["aci/" + aciname + ".asc"] = "file:" + strip_name(sig)

with open(output, "w") as f:
    json.dump(uploads, f)
