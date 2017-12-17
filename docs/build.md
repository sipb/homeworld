# Prerequisites

Install packages:

 * build-essential
 * sbuild
 * cpio
 * genisoimage

You will need to set up a sbuild chroot. (See https://github.com/sipb/homeworld/issues/80)

# Pull required libraries and images from upstream
    $ cd //building
    $ ./pull-upstream.sh

# Build helpers

    $ cd //building/build-helpers/helper-go/
    $ ./build.sh

    $ cd //building/build-helpers/helper-acbuild/
    $ ./build.sh

# Build and upload packages

    $ cd //building/build-debs/
    $ ./build-all.sh

To upload packages, ONLY IF YOU ARE SUPPOSED TO RELEASE YOUR CHANGES:

    $ cd //building/upload-debs/
    $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
    $ ./rebuild.sh

# Build containers

Install rkt from package (used in development environment for running builder containers):

    $ dpkg -i //building/build-debs/binaries/homeworld-rkt-<newest>.deb

To build containers:

    $ cd //building/build-acis/
    $ ./build-all.sh

To upload containers, ONLY IF YOU ARE SUPPOSED TO RELEASE YOUR CHANGES:

    $ cd //building/upload-acis/
    $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
    $ ./rebuild.sh
