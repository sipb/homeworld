#!/usr/bin/python

import sys
import subprocess
import urllib.parse
import os
import hashlib
import argparse

repo_url = "http://web.mit.edu/hyades/debian/"

argparser = argparse.ArgumentParser(description='Download a package from Homeworld repo')
argparser.add_argument('package_name', help="the name of the package to fetch")
argparser.add_argument('--verbose', '-v', action="store_true", help="make the output verbose")
args = argparser.parse_args()
verbose = args.verbose
package_name = args.package_name

if verbose:
    print("Looking for: ", package_name)
packages_file = open("Packages")
line = packages_file.readline().strip()
escape = 0

while not line == ("Package: " + package_name) and escape < 2:
    if line == "":
        escape += 1
    else:
        escape = 0
    line = packages_file.readline().strip()

url = None
expectedHash = None
while line is not "":
    if line.startswith("Filename: "):
        line = line.replace("Filename: ", "")
        url = repo_url + line
    if line.startswith("SHA256: "):
        expectedHash = line.replace("SHA256: ", "")

    if url is not None and expectedHash is not None:
        break

    line = packages_file.readline().strip()

if url is None:
    print("Unknown Package Name")
    exit(1)
if expectedHash is None:
    print("Cannot find SHA256 for package")
    exit(1)

filename = os.path.basename(urllib.parse.urlparse(url).path)
if verbose:
    print("Downloading file from url:", url)

subprocess.call(["curl", "-Os", url])

if verbose:
    print("Verifying file...")

downloadFile = open(filename, "rb")
actualHash = hashlib.sha256(downloadFile.read()).hexdigest()
if not expectedHash == actualHash:
    print("Verification failed!")
    exit(2)

if verbose:
   print("The downloaded package has the following filename:")
print(filename)

if verbose:
    print("Done!")




