#!/usr/bin/env python3

# takes in setup.conf and certificates.conf in the current directory
# and spits out a folder cluster-config/ with the generated configuration

import os
import sys

setup = "setup.conf"
certificates_in = "certificates.conf"
certificates_out = "certificates.list"
output = "cluster-config/"
cluster_config = "cluster.conf"

if os.path.exists(output):
	for file in os.listdir(output):
		os.remove(os.path.join(output, file))
else:
	os.mkdir(output)

def parse_setup(f):
	lines = [line.strip() for line in f if line.strip()]
	config = {}
	nodes = []
	for line in lines:
		if line[0] == '#':
			continue
		elif "=" in line:
			k, v = line.split("=")
			config[k] = v
		elif line[:7] in ("master ", "worker "):
			kind, hostname, ip = line.split()
			nodes.append((kind == "master", hostname, ip))
		else:
			raise Exception("Unrecognized line: %s" % line)
	return config, nodes

def generate_etcd_info(nodes):
	cluster = []
	endpoints = []
	for is_master, hostname, ip in nodes:
		if not is_master: continue
		cluster.append("{hostname}=https://{ip}:2380".format(hostname=hostname, ip=ip))
		endpoints.append("https://{ip}:2379".format(ip=ip))
	return ",".join(cluster), ",".join(endpoints)

with open(setup, "r") as f:
	config, nodes = parse_setup(f)

masters = [(hostname, ip) for ismaster, hostname, ip in nodes if ismaster]
workers = [(hostname, ip) for ismaster, hostname, ip in nodes if not ismaster]

config["ETCD_CLUSTER"], config["ETCD_ENDPOINTS"] = generate_etcd_info(nodes)

config["APISERVER_COUNT"] = len(masters)
print("TODO: use more than one apiserver for direct requests")
config["APISERVER"] = "https://{ip}:443".format(ip=masters[0][1])

with open(os.path.join(output, cluster_config), "w") as f:
	f.write("# generated from setup.conf automatically by compile-config.py\n")
	for kv in sorted(config.items()):
		f.write("%s=%s\n" % kv)

for ismaster, hostname, ip in nodes:
	with open(os.path.join(output, "node-%s.conf" % hostname), "w") as f:
		f.write("""# generated from setup.conf automatically by compile-config.py
HOST_NODE={hostname}
HOST_DNS={hostname}.{DOMAIN}
HOST_IP={ip}
SCHEDULE_WORK={schedule_work}
""".format(hostname=hostname, ip=ip, DOMAIN=config['DOMAIN'],
                           schedule_work=('false' if ismaster else 'true')))

with open(certificates_in, "r") as fin:
	with open(os.path.join(output, certificates_out), "w") as fout:
		fout.write("# generated from certificates.conf and setup.conf automatically by compile-config.py\n")
		nodelists = {"master": masters, "worker": workers, "all": masters + workers}
		for line in fin:
			line = line.strip()
			if not line or line[0] == '#':
				fout.write("\n")
			elif line.startswith("authority ") or line.startswith("shared-key "):
				fout.write(line + "\n")
			else:
				fout.write("# " + line + "\n")
				components = line.split(" ")
				needs_default_names = components[0] == "certificate"
				nodes_to_include = nodelists[components[1]] # must be master, worker, all
				for hostname, ip in nodes_to_include:
					ncomp = components[:]
					ncomp[1] = "%s.%s" % (hostname, config["DOMAIN"])
					if needs_default_names:
						ncomp.append("ip:%s" % ip)
						ncomp.append("dns:%s" % hostname)
						ncomp.append("dns:%s.%s" % (hostname, config["DOMAIN"]))
					fout.write(" ".join(ncomp) + "\n")

with open(os.path.join(output, "spin-up-all.sh"), "w") as f:
	f.write("#!/bin/bash\nset -e -u\n")
	f.write("# generated from setup.conf automatically by compile-config.py\n")
	f.write('cd "$(dirname "$0")"\n')
	for ismaster, hostname, ip in nodes:
		f.write("./spin-up.sh {host}\n".format(host=hostname))
	f.write("echo 'spun up all nodes!'\n")

os.chmod(os.path.join(output, "spin-up-all.sh"), 0o755)

with open(os.path.join(output, "spin-up.sh"), "w") as f:
	f.write("#!/bin/bash\nset -e -u\n")
	f.write("# generated from setup.conf automatically by compile-config.py\n")
	f.write('cd "$(dirname "$0")"\n')
	f.write("HOST=$1\n")
	f.write('if [ ! -e "node-$HOST.conf" ]; then echo "could not find node config for $HOST"; exit 1; fi\n')
	f.write("echo \"uploading to $HOST...\"\n")
	f.write('scp "node-$HOST.conf" "root@$HOST.{domain}:/etc/hyades/local.conf"\n'.format(domain=config["DOMAIN"]))
	f.write('scp cluster.conf "root@$HOST.{domain}:/etc/hyades/cluster.conf"\n'.format(domain=config["DOMAIN"]))
	f.write("echo \"uploaded to $HOST!\"\n")

os.chmod(os.path.join(output, "spin-up.sh"), 0o755)

with open(os.path.join(output, "pkg-install-all.sh"), "w") as f:
	f.write("#!/bin/bash\nset -e -u\n")
	f.write("# generated from setup.conf automatically by compile-config.py\n")
	f.write('cd "$(dirname "$0")"\n')
	for ismaster, hostname, ip in nodes:
		f.write('./pkg-install.sh {host}\n'.format(host=hostname))
	f.write("echo 'deployed to all nodes!'\n")

os.chmod(os.path.join(output, "pkg-install-all.sh"), 0o755)

with open(os.path.join(output, "pkg-install.sh"), "w") as f:
	f.write("#!/bin/bash\nset -e -u\n")
	f.write("# generated from setup.conf automatically by compile-config.py\n")
	f.write('cd "$(dirname "$0")"\n')
	f.write("HOST=$1\n")
	f.write("echo \"deploying to $HOST...\"\n")
	f.write("ssh \"root@$HOST.{domain}\" 'apt-get install homeworld-services'\n".format(domain=config["DOMAIN"]))
	f.write("echo \"deployed to $HOST!\"\n")

os.chmod(os.path.join(output, "pkg-install.sh"), 0o755)

print("Generated!")
