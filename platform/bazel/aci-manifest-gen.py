import sys
import json


def parse_port(port):
    # metrics:tcp:80
    assert port.count(":") == 2
    name, protocol, count = port.split(":")
    assert protocol in ("tcp", "udp")
    return {
        "name": name,
        "protocol": protocol,
        "port": int(count),
    }


with open(sys.argv[1], "r") as f:
    version = f.read().strip()
aciname = sys.argv[2]
exc = sys.argv[3:]
excid = exc.index("--")
ports = [parse_port(port) for port in exc[excid+1:]]
exc = exc[:excid]

print(json.dumps({
    "acKind": "ImageManifest",
    "acVersion": "0.8.11",
    "name": aciname,
    "labels": [
        {
            "name": "arch",
            "value": "amd64"
        },
        {
            "name": "os",
            "value": "linux"
        },
        {
            "name": "version",
            "value": version,
        },
    ],
    "app": {
        "exec": exc,
        "user": "0",
        "group": "0",
        "ports": ports,
    },
}))
