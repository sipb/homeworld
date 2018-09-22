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

# Getting Started

For information on building and deploying Hyades, check the 'docs' subdirectory.

# Contact

Project lead: cela. Contact over zephyr (-c hyades) or email @mit.edu.

test
