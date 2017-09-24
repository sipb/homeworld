#!/usr/bin/python3.6

import re
import hashlib
import urllib.request
import subprocess
import os

with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/Release.gpg') as response:
    file = open("Release.gpg", "w")
    file.write(response.read().decode('utf-8'))
    file.close()

with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/Release') as response:
    file = open("Release", "w")
    file.write(response.read().decode('utf-8'))
    file.close()

script_dir = os.path.dirname(os.path.abspath(__file__))
gpg_verify_exit_code = subprocess.call(["gpg", "--no-default-keyring", "--keyring", script_dir + "/../../build-debs/homeworld-apt-setup/homeworld-archive-keyring.gpg",
                 "--verify", "Release.gpg", "Release"])

if gpg_verify_exit_code:
    print("Failed to verify the Homeworld repo Release file")
    exit(1)

release_file = open('Release', 'r')
engaged = False
sha256_hash = None
for line in release_file:
    if line == 'SHA256:\n':
        engaged = True
        continue
    if engaged:
        if line[0] == ' ':
            line2 = re.sub(r"^(\w+) \d+ ([^ ]+)$", r"\2", line.strip())
            if line2 == 'main/binary-amd64/Packages':
                sha256_hash = re.sub(r"^(\w+) \d+ ([^ ]+)$", r"\1", line.strip())
                print('Found Packages SHA-256 Hash:', sha256_hash)
        else:
            engaged = False
            break
release_file.close()

if sha256_hash is None:
    print("Failed to extract SHA-256 Hash for Homeworld's repo Packages. Aborting...")
    exit(1)

Packages = None
with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/main/binary-amd64/Packages') as response:
    Packages = response.read()

if Packages is None:
    print("Failed to fetch Homeworld's Packages file from repo. Aborting...")
    exit(1)

packages_hash = hashlib.sha256(Packages).hexdigest()
if not packages_hash == sha256_hash:
    print("Packages file verification failed. Aborting...")
    exit(1)
else:
    print("Verified Packages file from repo")

packages_file = open("Packages", "w")
packages_file.write(Packages.decode('utf-8'))
packages_file.close()

print("Packages file saved as Packages.")
print("Done!")
