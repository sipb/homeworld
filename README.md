# Hyades Provisioning

# Subfolders

Top-level:

 * building: code and scripts for creating binaries and uploading them to repositories
 * docs: information on building and deploying a cluster
 * design: (incomplete) design documents for the system.

In more detail:

 * building/build-helpers: build scripts for intermediate tools (i.e. Go)
 * building/build-acis: container build scripts
 * building/build-debs: debian package build scripts
 * building/upstream: vendored upstream source packages (stored in a separate Git repository; see pull-upstream.sh)
 * building/upload-acis: uploading containers to the registry
 * building/upload-debs: uploading debian packages to the repository

# Repository Security

All commits in this repository are signed with GPG:

    pub   rsa4096 2016-10-12 [SC] [expires: 2017-10-12]
          EEA3 1BFF 4443 04AB B246  A0B6 C634 D042 0F82 5B91
    uid           [ultimate] Cel A. Skeggs <cela [at] mit [dot] edu>

(Obviously, you should be verifying this key out-of-band, not by checking it
against this page.)

Write access on GitHub is restricted to the hyades-provisioning team.

These security measures exist due to scripts from this repository being used on
trusted systems with /root kerberos tickets or other important auth keys.

# Contributing Guidelines

Every commit in this repository *MUST* be signed by a trusted developer. You
will need a PGP key in the web of trust, or at least a PGP key that the main
maintainer can trust. Before your PRs are merged, you will need to rebase all
of your commits against the current master branch and ensure that each of them
is signed. Consider combining small and pointless commits into larger commits.

## Guidelines for Go code

Every commit with Go code should be formatted with gofmt if possible. If not,
sequences of commits should be formatted at the end.

If the code is part of a security-critical component, there should be
reasonably complete unit tests before merging.

# Contact

Project lead: cela. Contact over zephyr (-c hyades) or email @mit.edu.
