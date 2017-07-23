#!/usr/bin/env python3

# takes in setup.conf and certificates.conf in the current directory
# and spits out a folder cluster-config/ with the generated configuration

import os
import sys
import shlex
import contextlib

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
	dns = []
	for line in lines:
		if line[0] == '#':
			continue
		elif "=" in line:
			k, v = line.split("=")
			config[k] = v
		elif line[:7] in ("master ", "worker "):
			kind, hostname, ip = line.split()
			nodes.append((kind == "master", hostname, ip))
		elif line.startswith("bootstrap-dns "):
			kind, domain, ip = line.split()
			dns.append((domain, ip))
		else:
			raise Exception("Unrecognized line: %s" % line)
	return config, nodes, dns

def generate_etcd_info(nodes):
	cluster = []
	endpoints = []
	for is_master, hostname, ip in nodes:
		if not is_master: continue
		cluster.append("{hostname}=https://{ip}:2380".format(hostname=hostname, ip=ip))
		endpoints.append("https://{ip}:2379".format(ip=ip))
	return ",".join(cluster), ",".join(endpoints)

with open(setup, "r") as f:
	config, nodes, dnses = parse_setup(f)

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

def begin_script(filename):
	filename = os.path.join(output, filename)
	out = open(filename, "w")
	try:
		os.chmod(filename, 0o755)
		out.write("#!/bin/bash\nset -e -u\n")
		out.write("# generated from setup.conf automatically by compile-config.py\n")
		out.write('cd "$(dirname "$0")"\n')
	except:
		out.close()
		os.remove(filename)
		raise
	return out

with begin_script("start-all.sh") as f:
	f.write("echo 'starting etcd on each master node'\n")
	first_master = None
	for ismaster, hostname, ip in nodes:
		if ismaster:
			if first_master is None:
				first_master = hostname
			f.write("echo 'for {host}'\n".format(host=hostname))
			f.write("ssh root@{host}.{domain} /usr/lib/hyades/start-master-etcd.sh\n".format(host=hostname,domain=config["DOMAIN"]))
	f.write("sleep 1\n")
	if first_master is None:
		f.write("echo 'not initializing flannel config'\n")
	else:
		f.write("echo 'initializing flannel config'\n")
		f.write("ssh root@{host}.{domain} /usr/lib/hyades/init-flannel.sh\n".format(host=first_master,domain=config["DOMAIN"]))
	f.write("echo 'starting other services on master nodes'\n")
	for ismaster, hostname, ip in nodes:
		if ismaster:
			f.write("echo 'for {host}'\n".format(host=hostname))
			f.write("ssh root@{host}.{domain} /usr/lib/hyades/start-master.sh\n".format(host=hostname,domain=config["DOMAIN"]))
	f.write("sleep 1\n")
	f.write("echo 'starting worker nodes'\n")
	for ismaster, hostname, ip in nodes:
		if not ismaster:
			f.write("echo 'for {host}'\n".format(host=hostname))
			f.write("ssh root@{host}.{domain} /usr/lib/hyades/start-worker.sh\n".format(host=hostname,domain=config["DOMAIN"]))
	f.write("echo 'started all nodes!'\n")

@contextlib.contextmanager
def begin_host_script(stem):
	with begin_script(stem + "-all.sh") as f:
		for ismaster, hostname, ip in nodes:
			f.write("./{stem}.sh {host}\n".format(stem=stem, host=hostname))
		f.write("echo 'Completed {stem} on all nodes!'\n".format(stem=stem))
	with begin_script(stem + ".sh") as f:
		f.write("HOST=$1\n")
		f.write('echo "Running {stem} on $HOST..."\n'.format(stem=stem))
		yield f
		f.write('echo "Finished {stem} on $HOST!"\n'.format(stem=stem))

with begin_host_script("pkg-install") as f:
	f.write("ssh \"root@$HOST.{domain}\" 'apt-get update && apt-get upgrade -y && apt-get install -y homeworld-services'\n".format(domain=config["DOMAIN"]))

if dnses:
	with begin_host_script("dns-bootstrap-add") as f:
		f.write("./dns-bootstrap-remove.sh \"$HOST\"\n") # make sure we don't double-add
		for domain, ip in dnses:
			remote_command = "echo '{ip}\t{domain} # AUTO-HOMEWORLD-BOOTSTRAP' >>/etc/hosts".format(ip=ip, domain=domain)
			remote_command = shlex.quote(remote_command)
			f.write('ssh "root@$HOST.{domain}" {command}\n'.format(domain=config["DOMAIN"], command=remote_command))

	with begin_host_script("dns-bootstrap-remove") as f:
		for domain, ip in dnses:
			remote_command = "grep -vF 'AUTO-HOMEWORLD-BOOTSTRAP' /etc/hosts >/etc/hosts.new && mv /etc/hosts.new /etc/hosts".format(ip=ip, domain=domain)
			remote_command = shlex.quote(remote_command)
			f.write('ssh "root@$HOST.{domain}" {command}\n'.format(domain=config["DOMAIN"], command=remote_command))

print("Generated!")
