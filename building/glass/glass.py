#!/usr/bin/env python3
"This module is the main entry point for the Glass build tool."
import argparse
import os

import aptbranch
import project
import upload

if not os.path.exists("/h/"):
    raise Exception("not currently running within the homeworld build chroot")

parser = argparse.ArgumentParser(description="perform a glass build")
parser.add_argument("-b", "--branch", default=aptbranch.get_env_branch(),
                    help="the apt branch to build this package for")
parser.add_argument("-d", "--debug", action="store_true", help="if an error occurs, drop to a debug shell")
parser.add_argument("-c", "--clean", action="store_true", help="delete any existing outputs of this build if found")
parser.add_argument("-r", "--rebuild", action="store_true", help="clean before building")
parser.add_argument("-u", "--upload", action="store_true", help="upload binaries as a new release, based on the URL embedded in the branch config")
parser.add_argument("-U", "--upload-only", action="store_true", help="only upload; do not build first")
parser.add_argument("projects", nargs="*", help="the directory of the package to build")

args = parser.parse_args()

# if they didn't give us a branch, and none was specified in the environment, we should throw an exception
if args.branch is None:
    print("Error: need to specify apt branch:")
    print("$ export HOMEWORLD_APT_BRANCH=<branch>")
    print("or include --branch=<branch> on command line for glass")
    print('Available apt branches:')
    for branch_name in aptbranch.Config.list_branches():
        print(branch_name)
    raise Exception("no apt branch specified")

branch_config = aptbranch.Config(args.branch)

if not args.upload_only:
    projects = [os.path.abspath(project) for project in args.projects or ["."]]

    os.chdir("/")  # to ensure that nothing uses relative paths when they shouldn't

    if args.clean and args.rebuild:
        raise Exception("cannot specify both --clean and --rebuild")

    projects = [project.Project(p) for p in projects]

    if args.clean or args.rebuild:
        any = False
        for p in projects:
            any |= p.clean(branch_config)
        if args.clean and not any:
            print(" ** nothing to clean")

    if not args.clean:
        for p in projects:
            p.run(branch_config, debug=args.debug)
else:
    print(" ** skipping any building")
    os.chdir("/")

if args.upload or args.upload_only:
    upload.upload(project.get_bindir(branch_config.name), branch_config)
