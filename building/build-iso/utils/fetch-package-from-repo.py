#!/usr/bin/python3

import sys
import subprocess
import urllib.parse
import os
import hashlib
import argparse

repo_url = "http://web.mit.edu/hyades/debian/"

argparser = argparse.ArgumentParser(description='Download a package from Homeworld repo, requires running verify-homeworld-repo.py first')
argparser.add_argument('package_name', help="the name of the package to fetch")
argparser.add_argument('--verbose', '-v', action="store_true", help="make the output verbose")
args = argparser.parse_args()
# Make key arguments into variables
verbose = args.verbose
package_name = args.package_name

if verbose:
    print('Looking for:', package_name)

url = None
expectedHash = None
with open('Packages') as packages_file:
   target_line = 'Package: ' + package_name
   file_name_target = 'Filename: '
   sha256_target = 'SHA256: '
   in_target_section = False
   for line in packages_file:
       line = line.strip()
	# Check if we are reading the section about the package we want info about, if so start caring about finding the needed package information
       if in_target_section:
           if line.startswith(file_name_target):
               filename = line[len(file_name_target):]
               url = repo_url + filename
           if line.startswith(sha256_target):
               expectedHash = line[len(sha256_target):]
           if len(line) == 0:
               break
       # We check for the target_line at the end of the current loop because this line *must* proceed what the relevant package information
       if line == target_line:
            in_target_section = True

if url is None:
    print('Unknown Package Name')
    exit(1)
if expectedHash is None:
    print('Cannot find SHA256 for package')
    exit(1)

filename = os.path.basename(urllib.parse.urlparse(url).path)
if verbose:
    print('Downloading file from url:', url)

subprocess.call(['curl', '-Os', '--', url])

if verbose:
    print('Verifying file...')
    print('Expected Hash:', expectedHash)

with open(filename, 'rb') as downloadFile:
    actualHash = hashlib.sha256(downloadFile.read()).hexdigest()
    # The extra spaces here are to line the hashes for display
    print('Actual Hash:  ', actualHash)
    if expectedHash != actualHash:
        print('Verification failed!')
        exit(2)

if verbose:
   print('The downloaded package has the following filename:')
# Only print the file name for the purposes of using this script in a bash script, if not in verbose mode
print(filename)

if verbose:
    print('Done!')
