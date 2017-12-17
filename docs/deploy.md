# Installing packages

To set up the apt repository:

    $ wget http://web.mit.edu/hyades/homeworld-apt-setup.deb
    $ wget http://web.mit.edu/hyades/homeworld-apt-setup.deb.asc
    $ gpg --verify homeworld-apt-setup.deb.asc
       ^^ IF THIS FAILS (or you haven't verified cela's key in person before),
          DELETE YOUR DOWNLOADS AND DO NOT CONTINUE
    $ sudo dpkg -i homeworld-apt-setup.deb

(You can also just build homeworld-apt-setup yourself.)

To install homeworld-admin-tools:

    $ sudo apt-get update
    $ sudo apt-get install homeworld-admin-tools

This will provide access to the 'spire' tool.

# Setting up a new cluster from scratch

## Setting up a workspace

You need to a set up an environment variable corresponding to a folder that can
store your cluster's configuration and authorities. Assuming that your disaster
recovery key (see below) is well-protected, this folder can be a publicly-
readable git repository.

WARNING: SUPPORT FOR GIT IS STILL IN PROGRESS; DO NOT USE IT UNLESS YOU KNOW
WHAT YOU ARE DOING. ESPECIALLY DO NOT CHECK IN ANY FILES THAT YOU ARE NOT 100%
CERTAIN ARE ENCRYPTED.

    $ export HOMEWORLD_DIR="$HOME/my-cluster"

## Setting up secure key storage

You need to choose a location to hold the disaster recovery key for your
cluster. If your cluster is for development purposes, it will suffice to store
it locally, but for production clusters it should be stored offline on
something like an encrypted USB drive.

    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"

This key will be used to encrypt the private authority keys.

**WARNING**: because gpg's `--passphrase-file` option is used, only the first
line from the file will be used as the key!

Recommended method of generating the passphrase:

    $ pwgen -s 160 1 >$HOMEWORLD_DISASTER

Make sure that you do not do this on a multi-user system, or that you've
otherwise protected the file that you're writing out from others.

## Configuring the cluster

Set up the configuration:

    $ spire config populate
    $ spire config edit

## Generating authority keys

    $ spire authority gen

## Acquiring upstream keys

 * Request a keytab from accounts@, if necessary
 * Import the keytab into the project:

       $ spire keytab import <hostname> <path-to-keytab>

 * Rotate the keytab (which includes upgrading its cryptographic strength):

       $ spire keytab rotate <hostname>
         # the following means invalidating current tickets:
       $ spire keytab delold <hostname>

 * If you are running your own homeworld bootstrap container registry, import the HTTPS key and certificate:

       $ spire https import homeworld.mit.edu ./homeworld.mit.edu.key ./homeworld.mit.edu.pem

Now you can consider putting this folder in Git, and then move on to 'Deploying a prepared cluster' below.

## Uploading to Git

SEE ABOVE FOR WARNINGS ABOUT USING GIT FOR THIS.

    $ cd $HOMEWORLD_DIR
    $ git init
    $ git add setup.yaml authorities.tgz keytab.*.crypt https.*    # be VERY CAREFUL about what you're adding!
    $ git commit
    $ git remote add origin ...
    $ git push -u origin master

# Deploying a prepared cluster

## Cloning existing cluster configuration

Skip this step if you're starting a new cluster.

To download existing configuration:

    $ export HOMEWORLD_DIR="$HOME/my-cluster"
    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"
    $ git clone git@github.com:sipb/hyades-cluster $HOMEWORLD_DIR

Make sure to verify that you have the correct commit hash, out of band.

## Set SSH configuration

Configure SSH so that it has the correct certificate authority in ~/.ssh/known_hosts for members of the cluster:

    $ spire access update-known-hosts

## Building the ISO

Now, create an ISO:

    $ spire iso gen preseeded.iso ~/.ssh/id_rsa.pub   # this SSH key is used for direct access during cluster setup

Now you should burn and/or upload preseeded.iso that you've just gotten, so
that you can use it for installing servers. Make a note of the password it
generated.

For the official homeworld servers:

    $ edit ~/.ssh/config
        Host toast
                HostName toastfs-dev.mit.edu
                User root
                GSSAPIAuthentication yes
                GSSAPIKeyExchange no
                GSSAPIDelegateCredentials no
    $ scp preseeded.iso toast:/srv/preseeded.iso

## Set up the supervisor operating system

 * Boot the ISO on the hardware
   - Select `Install`
   - Enter the IP address for the server (18.181.0.253 on our test infrastructure)
   - Wait a while
   - Enter "manual" for the bootstrap token (so that your SSH keys will work)
 * Log into the server directly with your SSH keys
   - Verify the host keys based on the text printed before the login console

## Setting up the supervisor node

Set up the keysystem:

    $ spire seq keysystem

Set up SSH:

      # if this fails, you might need to make sure you don't have any stale kerberos tickets
    $ spire seq ssh
      # (this command includes the automatic execution of `spire access ssh`)

## Set up each node's operating system

Request bootstrap tokens:

    $ spire infra admit-all

Boot the ISO on each piece of hardware
   - Select `Install`
   - Enter the IP address for the server
   - Wait a while
   - Enter the bootstrap token

Confirm that all of the servers came up properly (and requested their keys
correctly):

    $ spire verify online

## Core cluster bringup

Bring up the core cluster:

    $ spire seq core

If and only if you're hosting the containers for core cluster services on the
cluster itself:

    $ spire seq registry

## Core cluster service: flannel

Deploy flannel into the cluster:

    $ spire deploy flannel

Wait a bit for propagation... (if this doesn't work, keep trying for a bit)

    $ spire verify flannel-run
    $ spire verify flannel-ping

## Core cluster service: dns-addon

Deploy dns-addon into the clustesr:

    $ spire deploy dns-addon

Wait a bit for propagation... (if this doesn't work, keep trying for a bit)

    $ spire verify dns-addon-run
    $ spire verify dns-addon-query

## Finishing up

The cluster should now be ready!
