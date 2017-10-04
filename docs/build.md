Prerequisites:

Install packages:

 * build-essential
 * sbuild
 * cpio
 * genisoimage

You will need to set up a sbuild chroot. (See https://github.com/sipb/homeworld/issues/80)

First, build helpers:

 $ cd //building/build-helpers/helper-go/
 $ ./build.sh

 $ cd //building/build-helpers/helper-acbuild/
 $ ./build.sh

Next, build packages:

 $ cd //building/build-debs/
 $ ./build-all.sh

Next, upload packages:

 $ cd //building/upload-debs/
 $ aklog sipb    # authenticate to the SIPB AFS cell, if necessary
 $ ./rebuild.sh

Next, install rkt from package (used in development environment for running builder containers):

 $ dpkg -i //building/build-debs/binaries/homeworld-rkt-<newest>.deb

Next, build containers:

 $ cd //building/build-acis/
 $ ./build-all.sh
