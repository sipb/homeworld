import subprocess
import os

gpg_output = subprocess.check_output(["gpg", "--list-public-keys", "--with-fingerprint", "--with-colons"]).splitlines()
fpr_lines = [line for line in gpg_output if line.startswith("fpr:")]

if len(fpr_lines) == 0:
	raise Exception("no gpg public key fingerprints found")

if len(fpr_lines) > 1:
	raise Exception("multiple gpg public key fingerprints found")

fpr_fields = fpr_lines[0].split(":")
if len(fpr_fields) < 10:
	raise Exception("not enough fields in fingerprint data")

fpr = fpr_fields[9]

with open(".jenkins/branches.yaml.in") as fin, open("building/apt-branch-config/branches.yaml", "w") as fout:
	fout.write(fin.read()
		.replace("$$_SIGNING_KEY_$$", fpr)
		.replace("$$_APT_URL_$$", os.environ["APT_URL"]))
