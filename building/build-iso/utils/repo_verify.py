#!/usr/bin/python3

import re
import hashlib
import urllib.request
import subprocess
import os
import sys

def verify_homeworld_repo(usePrintStatements):
    with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/Release.gpg') as response:
        with open('Release.gpg', 'wb') as file:
           file.write(response.read())

    with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/Release') as response:
        with open('Release', 'wb') as file:
           file.write(response.read())

    script_dir = os.path.dirname(os.path.abspath(__file__))
    null_file = open(os.devnull, 'w')
    gpg_verify_exit_code = subprocess.check_call(['gpg', '--no-default-keyring', '--keyring', script_dir + '/../../build-debs/homeworld-apt-setup/homeworld-archive-keyring.gpg', '--verify', 'Release.gpg', 'Release'], stderr=sys.stderr if usePrintStatements else null_file, stdout=sys.stdout if usePrintStatements else null_file)

    if gpg_verify_exit_code:
        if usePrintStatements:
            print('Failed to verify the Homeworld repo Release file')
        return (False, 'Failed to verify the Homeworld repo Release file')

    with open('Release', 'r') as release_file:
        in_sha256_section = False
        sha256_hash = None
        for line in release_file:
            if in_sha256_section:
                if line[0] == ' ':
                    line2 = re.sub(r'^(\w+) \d+ ([^ ]+)$', r'\2', line.strip())
                    if line2 == 'main/binary-amd64/Packages':
                        sha256_hash = re.sub(r'^(\w+) \d+ ([^ ]+)$', r'\1', line.strip())
                        if usePrintStatements:
                            print('Found Packages SHA-256 Hash:', sha256_hash)
                else:
                    break
            if line == 'SHA256:\n':
                in_sha256_section = True

    if sha256_hash is None:
        if usePrintStatements:
            print('Failed to extract SHA-256 Hash for Homeworld\'s repo Packages. Aborting...')
        return (False, 'Failed to extract SHA-256 Hash for Homeworld\'s repo Packages file')

    Packages = None
    with urllib.request.urlopen('http://web.mit.edu/hyades/debian/dists/homeworld/main/binary-amd64/Packages') as response:
        Packages = response.read()

    if Packages is None:
        if usePrintStatements:
            print('Failed to fetch Homeworld\'s Packages file from repo. Aborting...')
        return (False, 'Failed to fetch Homeworld\'s Packages file from repo.')

    packages_hash = hashlib.sha256(Packages).hexdigest()
    if not packages_hash == sha256_hash:
        if usePrintStatements:
            print('Packages file verification failed. Aborting...')
        return (False, 'Packages file verification failed')
    else:
        if usePrintStatements:
            print('Verified Packages file from repo')

    with open('Packages', 'wb') as packages_file:
        packages_file.write(Packages)
    
    if usePrintStatements:
        print('Packages file saved as Packages.')
        print('Finished Homeworld verification!')

    return (True, 'Success')

if __name__ == '__main__':
    verify_homeworld_repo()
