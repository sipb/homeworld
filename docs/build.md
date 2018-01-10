# Prerequisites

Install packages:

 * build-essential
 * debhelper
 * debootstrap
 * sbuild
 * sudo
 * systemd-container

You will need to set up a sbuild chroot. (See https://github.com/sipb/homeworld/issues/80)

That will let you build everything as described here.  If you also want to be able to build spire outside of the sbuild chroot (not described here), install these packages too:

 * libarchive-tools
 * python3-yaml
 * zip

# Create a personal apt repository

Set the apt branch environment variable (you will have to do this every session):

    $ export HOMEWORLD_APT_BRANCH=<username>/<branch>

Generate a new key to sign the repository with. Using a regular long-lived key here is discouraged. You will need to use gpg2, since reprepro uses gpgme.

    $ gpg2 --full-gen-key

Do `gpg2 --list-keys --keyid-format long` to find the ID of the key you have just generated. Add an entry to `signing-keys`:

    $ echo "${HOMEWORLD_APT_BRANCH} <key-id>" >> building/setup-apt-branch/signing-keys

Import the default apt repository signing key:

    $ gpg2 --import building/upload-debs/default-repo-signing-key.gpg

If you would like to base your build on a different upstream branch, update `building/upload-debs/conf/updates` with the details of that upstream branch.

# Pull required libraries and images from upstream

    $ cd building
    $ ./pull-upstream.sh
    $ cd ..

# Build helpers

    $ cd building/build-helpers/helper-go
    $ ./build.sh
    $ cd ../../..

    $ cd building/build-helpers/helper-acbuild
    $ ./build.sh
    $ cd ../../..

# Build and upload packages

    $ cd building/build-debs
    $ ./build-all.sh
    $ cd ../..

To upload packages, ONLY IF YOU ARE SUPPOSED TO RELEASE YOUR CHANGES:

    $ cd building/upload-debs
        # Note that you will need Kerberos tickets
        # (generate them with kinit) to access AFS.
    $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
    $ ./rebuild.sh
    $ cd ../..

# Build containers

Install rkt from package (used in development environment for running builder containers):

    $ dpkg -i building/build-debs/binaries/homeworld-rkt_<newest>.deb
    $ apt install -f    # if needed to resolve dependencies

To build containers:

    $ cd building/build-acis
    $ ./build-all.sh
    $ cd ../..

To upload containers, ONLY IF YOU ARE SUPPOSED TO RELEASE YOUR CHANGES:

    $ cd building/upload-acis
        # Note that you will need Kerberos tickets
        # (generate them with kinit) to access AFS.
    $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
    $ ./rebuild.sh
    $ cd ../..
