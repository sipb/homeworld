# Prerequisites

We only officially support builds on a Debian 9 (Stretch) or Debian 10 (Buster) host system.
You can try something else, but if it breaks, you get to keep all the pieces.

Install the following on your host system:

 * build-essential
 * debootstrap
 * realpath
 * sudo
 * systemd-container
 * gpg
 * curl

# Setting up

Set up your build chroot:

    $ export HOMEWORLD_CHROOT="$HOME/homeworld-chroot"     # this can be any directory you choose
    $ ./build-chroot/create.sh

You might consider adding the variable declaration to your ~/.profile.

If you're using RHOMBI, you should place your chroots in /opt/cluster, which is a larger disk.

    $ export HOMEWORLD_CHROOT="/opt/cluster/$USER/build-chroot"

(Ask for your directory to be created -- or create it yourself -- if it doesn't exist.)

# Setting up a build branch

You can either set up a Google Cloud Storage bucket to serve your uploaded build artifacts, or you can
set up an rsyncable directory. RHOMBI is already set up to serve artifacts, so if you have access to
that machine, you should use that.

## Using rsync on RHOMBI

First, you'll need a directory under `/var/www/html/homeworld-apt/` to be created for your user.

You'll need to come up with a branch name, like `test1`. You probably want to create a directory to
serve the artifacts for that particular branch:

    $ mkdir /var/www/html/homeworld-apt/$USER/test1

Then, you should generate a PGP signing key for uploading:

    $ gpg --full-gen-key
    Expire: 0 = key does not expire
    Real name: Homeworld Development Signing Key
    Email address: sipb-hyades-root@mit.edu
    Comment: (YOUR KERBEROS USERNAME)

Create the branch config from the template:

    $ (cd platform/upload && cp branches.yaml.example branches.yaml)

Run `gpg --list-keys --keyid-format none` to find the full-length fingerprint of the key you generated.

Now edit the branches.yaml to include the branch name, and to have the right upload and download links:

    branches:
      - name: test1
        signing-key: 1ECADC3E4D93A543D39639621BBDA050BADBB86A
        download: http://rhombi.mit.edu/homeworld-apt/cela/test1
        upload:
          method: rsync
          rsync-target: /var/www/html/homeworld-apt/cela/test1

Now, you can start your build.

## Using Google Cloud Storage

You'll need to start by creating a Google Cloud Storage bucket. You should set it up to serve files
with a public default ACL:

    $ gsutil defacl ch -u AllUsers:R gs://<name of bucket>

Create a service account with the Storage Object Admin permission on the bucket's project.
Put the service account's private key JSON file into a file named `boto-key` in the `homeworld` directory.
(That is, put the file in the root directory of the repository.)

You should generate a PGP signing key for uploading:

    $ gpg --full-gen-key
    Expire: 0 = key does not expire
    Real name: Homeworld Development Signing Key
    Email address: sipb-hyades-root@mit.edu
    Comment: (YOUR KERBEROS USERNAME)

Create the branches config:

    $ (cd platform/upload && cp branches.yaml.example branches.yaml)

Run `gpg --list-keys --keyid-format none` to find the full-length fingerprint of the key you generated.

Now, come up with a branch name, like `test1`, and edit the branches.yaml to include a branch of the
upload style you want to use, similar to the following:

    branches:
      - name: test1
        signing-key: 1ECADC3E4D93A543D39639621BBDA050BADBB86A
        download: http://mybucket.storage.googleapis.com/test1
        upload:
          method: google-cloud-storage
          gcs-target: gs://mybucket/test1

Note that you will need upload access to the relevant Google Cloud Storage bucket to actually upload to this branch.

# Launching a build

To set the build branch to use:

    $ echo "<branch>" >platform/upload/BRANCH_NAME

(Note that `BRANCH_NAME` is a literal; you should not replace it with the branch name.)

To enter the build chroot, run:

    $ ./build-chroot/enter.sh

(Do not use enter-ci.sh; it is unstable and only for use in continuous integration environments.)

Now, launch the build and upload process:

    $ cd platform
    $ bazel run //upload

If you just want to build without uploading:

    $ bazel build //upload

This will automatically run all of the required build steps.

Bazel will automatically handle incremental changes, so if one of these commands completes successfully, you can always rerun it and have nothing change.

Congratulations! You are ready to [deploy your very own Homeworld cluster](deploy.md).
