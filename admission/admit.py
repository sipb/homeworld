#!/usr/bin/env python3
import sys
import tempfile
import threading
import subprocess
import os
import socket
import hashlib
import base64
import http.server
import ssl

if len(sys.argv) < 5:
	print("Usage: admit.sh <hostname> <host_ca-key> <user_ca-pubkey> <sshd_config>", file=sys.stderr)
	sys.exit(1)

hostname, host_ca, user_ca, sshd_config = sys.argv[1:5]

assert not host_ca.endswith(".pub")
assert user_ca.endswith(".pub")
assert os.path.basename(sshd_config) == "sshd_config"

if not os.path.exists(host_ca):
	print("Host certificate authority does not exist.", file=sys.stderr)
	sys.exit(1)
if not os.path.exists(user_ca):
	print("User certificate authority does not exist.", file=sys.stderr)
	sys.exit(1)
if not os.path.exists(sshd_config):
	print("sshd configuration does not exist.", file=sys.stderr)
	sys.exit(1)

with open(os.path.join(os.path.dirname(__file__), "setup.sh"), "rb") as f:
	setup_script = f.read()

def get_local_ip():
	s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
	try:
		s.connect(('8.8.8.8', 1))  # doesn't actually send anything
		return s.getsockname()[0]
	finally:
		s.close()

local_ip = get_local_ip()
secure_token = base64.b64encode(os.urandom(10)).rstrip(b"=")

def stop_soon(server):
	print("will stop")
	threading.Timer(0.1, lambda: (server.shutdown(), print("perform stop"))).start()

class InitialRequestHandler(http.server.BaseHTTPRequestHandler):
	def do_GET(self):
		if self.client_address[0] != socket.gethostbyname(hostname):
			print("Got connection from wrong client", self.client_address)
			self.send_error(403)
			return
		if self.path == "/setup":
			self.send_response(200)
			self.send_header("Content-type", "text/plain")
			self.send_header("Content-length", len(setup_script))
			self.send_header("Last-Modified", self.date_time_string())
			self.end_headers()
			self.wfile.write(setup_script)
			stop_soon(self.server)
		else:
			self.send_error(404)

class SecondaryRequestHandler(http.server.BaseHTTPRequestHandler):
	def do_GET(self):
		if self.client_address[0] != socket.gethostbyname(hostname):
			print("Got connection from wrong client", self.client_address)
			self.send_error(403)
			return
		if self.path.startswith("/sign/"):
			pubkey_to_sign = self.path[len("/sign/"):].replace("%20"," ").encode() + b"\n"
			print("======================= VERIFY ========================")
			print(hashlib.sha256(pubkey_to_sign).hexdigest())
			print("======================= VERIFY ========================")
			if input("Correct? (y/n) ").strip()[:1].lower() == "y":
				with tempfile.TemporaryDirectory(prefix="hyades-admit-") as tmp:
					pubkeyname = os.path.join(tmp, "key.pub")
					with open(pubkeyname, "wb") as pubkey:
						pubkey.write(pubkey_to_sign)
						pubkey.flush()
					subprocess.check_call(["ssh-keygen", "-s", host_ca, "-h", "-I", "hyades_host_" + hostname, "-Z", hostname, "-V", "-1w:+30w", pubkeyname])
					certname = os.path.join(tmp, "key-cert.pub")
					with open(certname, "rb") as cert:
						certdata = cert.read()
				self.send_response(200)
				self.send_header("Content-type", "text/plain")
				self.send_header("Content-length", len(certdata))
				self.send_header("Last-Modified", self.date_time_string())
				self.end_headers()
				self.wfile.write(certdata)
			else:
				self.send_error(403)
		elif self.path == "/finish":
			self.send_response(200)
			self.send_header("Content-type", "text/plain")
			self.send_header("Content-length", len("stopping...\n"))
			self.send_header("Last-Modified", self.date_time_string())
			self.end_headers()
			self.wfile.write(b"stopping...\n")
			stop_soon(self.server)
		else:
			self.send_error(404)

PORT=20557

with tempfile.NamedTemporaryFile() as privatekey:
	with tempfile.NamedTemporaryFile() as certificate:
		subprocess.check_call(["openssl", "genrsa", "-out", privatekey.name, "2048"])
		subprocess.check_call(["openssl", "req", "-new", "-x509", "-key", privatekey.name, "-out", certificate.name, "-days", "1", "-subj", "/CN=" + local_ip])

		setup_script = setup_script.replace(b"{{SERVER CERTIFICATE}}", certificate.read())
		setup_script = setup_script.replace(b"{{SERVER IP}}", ("%s:%d" % (local_ip, PORT)).encode())
		with open(user_ca, "rb") as f:
			setup_script = setup_script.replace(b"{{SSH CA}}", f.read())
		with open(sshd_config, "rb") as f:
			setup_script = setup_script.replace(b"{{SSHD CONFIG}}", f.read())
		assert setup_script.count(b"_HERE") == 6  # avoid confusion of terminators
		setup_hash = hashlib.sha256(setup_script).hexdigest()

		print("Run on the target system:")
		print("    $ wget {ip}:{port}/setup".format(ip=local_ip, port=PORT))
		print("    $ sha256sum setup")
		print("    {hash}  setup".format(hash=setup_hash))
		print("    $ # after verifying hash")
		print("    $ bash setup")
		print("    # and compare the output to our output ...")

		# first, without SSL, to let them download /setup
		httpd = http.server.HTTPServer(('', PORT), InitialRequestHandler)
		httpd.serve_forever(poll_interval=0.1)
		httpd.server_close()
		print("/setup downloaded; moving to phase two")
		# then again, with SSL, to receive the request from the script
		httpd = http.server.HTTPServer(('', PORT), SecondaryRequestHandler)
		httpd.socket = ssl.wrap_socket(httpd.socket, certfile=certificate.name, keyfile=privatekey.name, server_side=True, ciphers="TLSv1.2")
		httpd.serve_forever(poll_interval=0.1)
		httpd.server_close()
