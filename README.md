# The Homeworld Project

Homeworld is a cluster orchestration system, built for the [Hyades cluster](http://hyades.mit.edu/).

# Subfolders

Top-level:

 * building: code and scripts for creating binaries and uploading them to repositories
 * docs: information on building and deploying a cluster
 * design: (incomplete) design documents for the system.

More detail:

 * building/binaries: gitignore'd directory containing built Homeworld binaries
 * building/components: build scripts and Homeworld-specific source code for the different components of Homeworld
 * building/glass: the build system used by Homeworld
 * building/setup-apt-branch: files for the deployment branch system
 * building/upstream: vendored upstream source packages (stored in a separate Git repository; see pull-upstream.sh)
 * building/upstream-check: scripts for verifying and redownloading upstream source code files

# Repository Security

All commits in this repository are signed with GPG:

    pub   rsa4096 2016-10-12 [SC] [expires: 2017-10-12]
          EEA3 1BFF 4443 04AB B246  A0B6 C634 D042 0F82 5B91
    uid           [ultimate] Cel A. Skeggs <cela [at] mit [dot] edu>

(Obviously, you should be verifying this key out-of-band, not by checking it
against this page.)

# Contributing Guidelines

Every commit in this repository *MUST* be signed by a trusted developer. You
will need a PGP key in the web of trust, or at least a PGP key that the main
maintainer can trust. Before your PRs are merged, you will need to rebase all
of your commits against the current master branch and ensure that each of them
is signed. Consider combining small and pointless commits into larger commits.

# Contact

Project lead: cela. Contact over zephyr (-c hyades) or email @mit.edu.
