#!/usr/bin/python3

import sys
import repo_verify
import repo_fetch

repo_verification = repo_verify.verify_homeworld_repo(False)
if not repo_verification[0]:
    print(repo_verification[1])
    exit(1)

for package in sys.argv[1:]:
    repo_fetch.fetch_homeworld_package(package, False)
