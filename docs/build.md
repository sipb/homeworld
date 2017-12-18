# Prerequisites

Install packages:

 * build-essential
 * debhelper
 * debootstrap
 * sbuild
 * sudo
 * systemd-container

You will need to set up a sbuild chroot. (See https://github.com/sipb/homeworld/issues/80)

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
    $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
    $ ./rebuild.sh
    $ cd ../..
