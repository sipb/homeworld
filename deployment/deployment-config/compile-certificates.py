#!/usr/bin/env python3

# takes in a parameter which specifies a certificates.list to use
# and spits out a certificate-plan/ folder in the current directory

import os
import sys

if len(sys.argv) != 3:
	print("Usage: compile-certificates.py <certificates.list> <secret-dir>")
	sys.exit(1)

output = "certificate-plan"
AUTHORITY_LIFETIME = 90
AUTHORITY_KEYSIZE = 2048
NODE_LIFETIME = 90
NODE_KEYSIZE = 2048
SHARED_KEYSIZE = 2048

if os.path.exists(output):
	for file in os.listdir(output):
		os.remove(os.path.join(output, file))
else:
	os.mkdir(output)

certlist, secretdir = sys.argv[1:]
secretdir = os.path.realpath(secretdir)
if "/afs/sipb.mit.edu/" in secretdir:
	secretdir = secretdir.replace("/afs/sipb.mit.edu/", "/afs/sipb/")

with open(certlist, "r") as f:
	lines = [line.strip() for line in f if line.strip() and not line.lstrip().startswith("#")]

authorities = []
placements = []
private_keys = []
shared_keys = []
shared_placements = []
certificates = []

for line in lines:
	components = line.split(" ")
	if components[0] == "authority":
		assert len(components) == 2
		authorities.append(components[1])
	elif components[0] == "place-authority":
		assert len(components) == 4
		placements.append(tuple(components[1:]))
	elif components[0] == "private-key":
		assert len(components) == 4
		private_keys.append(tuple(components[1:]))
	elif components[0] == "shared-key":
		assert len(components) == 2
		shared_keys.append(components[1])
	elif components[0] == "place-shared":
		assert len(components) == 4
		shared_placements.append(tuple(components[1:]))
	elif components[0] == "certificate":
		assert len(components) >= 5
		main = tuple(components[1:5])
		names = [name.split(":") for name in components[5:]]
		ips = [v for k, v in names if k == "ip"]
		hostnames = [v for k, v in names if k == "dns"]
		assert len(ips) + len(hostnames) == len(names)
		certificates.append(main + (ips, hostnames))
	else:
		raise Exception("Unknown line: %s" % components)

def begin(filename):
	f = open(os.path.join(output, filename), "w")
	f.write("""#!/bin/bash
# generated from certificates.list automatically by compile-certificates.py
# filename: {filename}
# secret directory: {secretdir}
set -e -u

""".format(filename=filename, secretdir=secretdir))
	os.chmod(os.path.join(output, filename), 0o755)
	return f

with begin("authority-gen.sh") as f:
	f.write("echo 'generating authorities...'\n")
	for authority in authorities:
		base = "{secretdir}/authority-{authority}".format(secretdir=secretdir, authority=authority)
		f.write("""# for authority {authority}
if [ ! -e "{base}-key.pem" ]
then
	echo '    generating key for {authority}'
	openssl genrsa -out {base}-key.pem {keysize}
	openssl req -x509 -new -nodes -key {base}-key.pem -days {lifetime} -out {base}.pem -subj "/CN=hyades-authority-{authority}"
	echo '    generated authority!'
else
	echo '    skipping key generation for {authority}'
fi

""".format(authority=authority, base=base, lifetime=AUTHORITY_LIFETIME, keysize=AUTHORITY_KEYSIZE))
	f.write("echo 'generated all authorities!'\n")

with begin("authority-check.sh") as f:
	f.write("echo 'checking authorities...'\n")
	for authority in authorities:
		base = "{secretdir}/authority-{authority}".format(secretdir=secretdir, authority=authority)
		f.write("""# for authority {authority}
if [ ! -e "{base}-key.pem" ]
then
	echo 'could not find key for {authority}!'
	exit 1
else
	echo '    found key for {authority}'
fi

""".format(authority=authority, base=base))
	f.write("echo 'found all authorities!'\n")

authority_location_lookup = {}

with begin("authority-upload.sh") as f:
	dirs_by_host = {host: set() for host, authority, path in placements}
	for host, authority, path in placements:
		dirs_by_host[host].add(os.path.dirname(path))
	f.write("echo 'creating directories on hosts...'\n")
	for host, dirs in dirs_by_host.items():
		command = " && ".join("mkdir -p {dir}".format(dir=dir) for dir in dirs)
		f.write("echo '    {host}'\n".format(host=host))
		f.write('ssh root@{host} "{command}"\n'.format(host=host, command=command))
	f.write("echo 'now uploading authorities'\n")
	for host, authority, path in sorted(placements, key=lambda x: x[1]):
		authority_location_lookup[(host, authority)] = path
		key = "{secretdir}/authority-{authority}.pem".format(secretdir=secretdir, authority=authority)
		f.write("echo '    uploading {authority} to {host}'\n".format(authority=authority, host=host))
		f.write("scp {key} root@{host}:{path}\n".format(key=key, host=host, path=path))
	f.write("echo 'all authorities uploaded!'\n")

private_location_lookup = {}

with begin("private-gen.sh") as f:
	f.write("echo 'generating private keys...'\n")
	for host, kid, path in private_keys:
		private_location_lookup[(host, kid)] = path
		f.write("echo '    generating {key} on {host}'\n".format(key=kid, host=host))
		f.write("ssh root@{host} \"if [ ! -e {path} ]; then openssl genrsa -out {path} {keysize} && echo '    generated {key}'; else echo '    skipped {key}'; fi\"\n".format(key=kid, host=host, path=path, keysize=NODE_KEYSIZE))
	f.write("echo 'all private keys generated!'\n")

with begin("private-check.sh") as f:
	f.write("echo 'checking private keys...'\n")
	for host, kid, path in private_keys:
		private_location_lookup[(host, kid)] = path
		f.write("echo '    generating {key} on {host}'\n".format(key=kid, host=host))
		f.write("if ssh root@{host} \"[ -e {path} ]\"; then echo '    found {key} on {host}'; else echo 'could not find {key} on {host}!'; exit 1; fi\n".format(key=kid, host=host, path=path, keysize=NODE_KEYSIZE))
	f.write("echo 'found all private keys!'\n")

with begin("shared-gen.sh") as f:
	f.write("echo 'generating shared keys...'\n")
	for kid in shared_keys:
		kfile = "{secretdir}/shared-{key}.key".format(secretdir=secretdir, key=kid)
		f.write("""# for shared key {key}
if [ ! -e "{kfile}" ]
then
	echo '    generating shared key {key}'
	openssl genrsa -out {kfile} {keysize}
	echo '    generated shared key!'
else
	echo '    skipping shared key generation for {key}'
fi

""".format(key=kid, kfile=kfile, keysize=SHARED_KEYSIZE))
	f.write("echo 'generated all shared keys!'\n")

with begin("shared-check.sh") as f:
	f.write("echo 'checking shared keys...'\n")
	for kid in shared_keys:
		kfile = "{secretdir}/shared-{key}.key".format(secretdir=secretdir, key=kid)
		f.write("""# for shared key {key}
if [ ! -e "{kfile}" ]
then
	echo 'could not find shared key {key}!'
	exit 1
else
	echo '    found shared key {key}'
fi

""".format(key=kid, kfile=kfile))
	f.write("echo 'found all shared keys!'\n")

with begin("shared-upload.sh") as f:
	f.write("echo 'uploading shared keys...'\n")
	for host, kid, path in sorted(shared_placements, key=lambda x: x[1]):
		kfile = "{secretdir}/shared-{key}.key".format(secretdir=secretdir, key=kid)
		f.write("echo '    uploading {key} to {host}'\n".format(key=kid, host=host))
		f.write("scp {kfile} root@{host}:{path}\n".format(kfile=kfile, host=host, path=path))
	f.write("echo 'all shared keys uploaded!'\n")

with begin("certificate-gen-csrs.sh") as f:
	f.write("echo 'generating csrs...'\n")
	for host, authority, kid, path, ips, hostnames in certificates:
		f.write("echo '    generating csr for {key} against {authority} on {host}'\n".format(key=kid, host=host, authority=authority))
		private_path = private_location_lookup[(host, kid)] # will fail if the key does not exist on this host
		csr_path = "{secretdir}/csr-{host}-{authority}-{key}.csr".format(secretdir=secretdir, host=host, authority=authority, key=kid)
		openssl_command = "cat >/tmp/csr-{key}-{authority}.cnf && openssl req -new -key {private_path} -config /tmp/csr-{key}-{authority}.cnf -subj '/CN=hyades-key-{key}-{host}'".format(key=kid, authority=authority, host=host, private_path=private_path)
		f.write("ssh root@{host} \"{command}\" >{csr_path} <<EOCONFIG\n".format(command=openssl_command, csr_path=csr_path, host=host))
		f.write("""[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
""")
		for i, hostname in enumerate(hostnames, 1):
			f.write("DNS.%d = %s\n" % (i, hostname))
		for i, ip in enumerate(ips, 1):
			f.write("IP.%d = %s\n" % (i, ip))
		f.write("EOCONFIG\n")
		# TODO: code to validate the CSR against the configuration
	f.write("echo 'all csrs generated!'\n")

with begin("certificate-sign-csrs.sh") as f:
	f.write("echo 'signing csrs...'\n")
	for host, authority, kid, path, ips, hostnames in certificates:
		f.write("echo '    signing csr for {key} against {authority} on {host}'\n".format(key=kid, authority=authority, host=host))
		authbase = "{secretdir}/authority-{authority}".format(secretdir=secretdir, authority=authority)
		csr_path = "{secretdir}/csr-{host}-{authority}-{key}.csr".format(secretdir=secretdir, host=host, authority=authority, key=kid)
		cert_path = "{secretdir}/cert-{host}-{authority}-{key}.pem".format(secretdir=secretdir, host=host, authority=authority, key=kid)
		f.write("""cat >/tmp/csr-{host}-{key}-{authority}.cnf <<EOCONFIG
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
""".format(host=host, key=kid, authority=authority))
		for i, hostname in enumerate(hostnames, 1):
			f.write("DNS.%d = %s\n" % (i, hostname))
		for i, ip in enumerate(ips, 1):
			f.write("IP.%d = %s\n" % (i, ip))
		f.write("EOCONFIG\n")
		f.write("openssl x509 -req -in {csr_path} -CA {authbase}.pem -CAkey {authbase}-key.pem -CAcreateserial -out {cert_path} -days {lifetime} -extensions v3_req -extfile /tmp/csr-{host}-{key}-{authority}.cnf\n".format(csr_path=csr_path, authbase=authbase, cert_path=cert_path, lifetime=NODE_LIFETIME, host=host, key=kid, authority=authority))
	f.write("echo 'all csrs signed!'\n")

with begin("certificate-upload-certs.sh") as f:
	f.write("echo 'uploading certs...'\n")
	for host, authority, kid, path, ips, hostnames in certificates:
		f.write("echo '    uploading cert for {key} against {authority} on {host}'\n".format(key=kid, host=host, authority=authority))
		cert_path = "{secretdir}/cert-{host}-{authority}-{key}.pem".format(secretdir=secretdir, host=host, authority=authority, key=kid)
		f.write("scp {cert_path} root@{host}:{path}\n".format(cert_path=cert_path, path=path, host=host))
	f.write("echo 'all certs uploaded!'\n")

with begin("certify.sh") as f:
	f.write("echo 'handling all certificate operations (besides creating authorities)...'\n")
	for cmd in ["authority-check", "authority-upload", "private-gen", "shared-gen", "shared-upload", "certificate-gen-csrs", "certificate-sign-csrs", "certificate-upload-certs"]:
		f.write("./%s.sh\n" % cmd)
	f.write("echo 'all certificate operations handled!'\n")

print("Certificate scripts generated.")
