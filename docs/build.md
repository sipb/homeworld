# Prerequisites

We only officially support builds on a Debian 9 (Stretch) host system.
You can try something else, but if it breaks, you get to keep all the pieces.

Install the following on your host system:

 * build-essential
 * cpio
 * squashfs-tools
 * debootstrap
 * realpath
 * sudo
 * systemd-container
 * gpg

# Setting up

Set up your build chroot:

    $ export HOMEWORLD_CHROOT="$HOME/homeworld-chroot"     # this can be any directory you choose
    $ ./create-chroot.sh

You might consider adding the variable declaration to your ~/.profile.

Import the default branch signing key:

    $ gpg --import building/apt-branch-config/default-key.asc

Pull down the upstream dependencies for Homeworld:

    $ building/pull-upstream.sh

# Setting up a build branch

A build branch will, first and foremost, require a Google Cloud Storage bucket to upload into.
(Other providers are also planned for support.)

You should set up your bucket to serve files with a public default ACL:

    $ gsutil defacl ch -u AllUsers:R gs://<name of bucket>

Create a service account with the Storage Object Admin permission on the bucket's project.
Put the service account's private key JSON file into a file named `boto-key` in the `homeworld` directory.
(That is, put the file in the root directory of the repository.)

You can then use the bucket's public domain name to construct a build branch:

    branch format: <domain>/<subbranch>
    example: hyades-deploy.celskeggs.com/test3

You should also generate a PGP key for your branch:

    $ gpg --full-gen-key

Create the branches config:

    $ (cd building/apt-branch-config && cp branches.yaml.example branches.yaml)

Run `gpg --list-keys --keyid-format none` to find the full-length fingerprint of the key you have just generated, and add it to `branches.yaml`.
Remove the unnecessary `apt-url-prefix` line.

# Launching a build

To enter the build chroot, run:

    $ ./enter-chroot.sh

(Do not use enter-chroot-ci.sh; it is unstable and only for use in continuous integration environments.)

Inside the chroot, set your build branch:

    # export HOMEWORLD_APT_BRANCH=<domain>/<subbranch>

For example:

    # export HOMEWORLD_APT_BRANCH=hyades-deploy.celskeggs.com/test3

Note that you will need upload access to the relevant Google Cloud Storage bucket to actually upload to this URL.

Then, launch the build:

    # cd components
    # glass

This will automatically run all of the required build steps.

If you want to upload after the build completes, use the `--upload` option:

    # glass --upload

Congratulations! You are ready to deploy your very own Homeworld cluster.
