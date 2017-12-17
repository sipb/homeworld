# Prerequisites

If you're re-deploying the cluster for development, you will need:

* A Debian installation (or Debian VM) -- note that we do not support any other environments.
* The disaster recovery key.
* Access to [hyades-cluster](https://github.mit.edu/sipb/hyades-cluster), where we store the current cluster configuration. You will need to have set up SSH keys with github.mit.edu.
* Your Kerberos identity in the root-admins secion of ``setup.yaml``. If it isn't there, you can just add it in yourself.
* Access to toastfs-dev (the machine which hosts the development cluster). You will need a Kerberos root instance as a prerequisite to this.
* Any VNC viewer. For example, TightVNC (``sudo apt-get install xtightvncviewer``), but it has an annoying bug where it crashes on window resizes. [TigerVNC](https://github.com/TigerVNC/tigervnc/releases) also works, but requires manual installation.

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

**NOTE**: If you're re-deploying the cluster for development, follow [Deploying a prepared cluster](#deploying-a-prepared-cluster) instead. You might additionally want to regenerate authority keys (``spire authority gen``) -- but you'd need to push to hyades-cluster if you make any changes to the cluster configuration.

## Setting up a workspace

You need to a set up an environment variable corresponding to a folder that can store your cluster's configuration and authorities. Assuming that your disaster recovery key (see below) is well-protected, this folder can be a publicly-readable git repository.

WARNING: SUPPORT FOR GIT IS STILL IN PROGRESS; DO NOT USE IT UNLESS YOU KNOW WHAT YOU ARE DOING. ESPECIALLY DO NOT CHECK IN ANY FILES THAT YOU ARE NOT 100% CERTAIN ARE ENCRYPTED.

    $ export HOMEWORLD_DIR="$HOME/my-cluster"

## Setting up secure key storage

You need to choose a location to hold the disaster recovery key for your cluster. If your cluster is for development purposes, it will suffice to store it locally, but for production clusters it should be stored offline on something like an encrypted USB drive.

    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"

This key will be used to encrypt the private authority keys.

**WARNING**: because gpg's `--passphrase-file` option is used, only the first line from the file will be used as the key!

Recommended method of generating the passphrase:

    $ pwgen -s 160 1 >$HOMEWORLD_DISASTER

Make sure that you do not do this on a multi-user system, or that you've otherwise protected the file that you're writing out from others.

## Configuring the cluster

Set up the configuration:

    $ spire config populate
    $ spire config edit

## Generating authority keys

    $ spire authority gen

## Acquiring upstream keys

**WARNING**: If you're re-deploying the cluster for development, you should not be following this section. Critically, you should not rotate the keytab, or you'd need to distribute the new keytab to everyone. Follow [Deploying a prepared cluster](#deploying-a-prepared-cluster) instead.

 * Request a keytab from accounts@, if necessary
 * Import the keytab into the project:

```
$ spire keytab import <hostname> <path-to-keytab>
```

 * Rotate the keytab (which includes upgrading its cryptographic strength):

```
$ spire keytab rotate <hostname>
   # the following means invalidating current tickets:
$ spire keytab delold <hostname>
```

 * If you are running your own homeworld bootstrap container registry, import the HTTPS key and certificate:

```
$ spire https import homeworld.mit.edu ./homeworld.mit.edu.key ./homeworld.mit.edu.pem
```

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

## Cloning an existing cluster configuration

To download existing configuration:

    $ export HOMEWORLD_DIR="$HOME/my-cluster"
    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"
    $ git clone git@github.mit.edu:sipb/hyades-cluster $HOMEWORLD_DIR

Make sure to verify that you have the correct commit hash, out of band.

## Extracting upstream keys

For the development cluster, we already have upstream keys. Extract them from the configuration like so:

    $ spire keytab export egg-sandwich egg-keytab
    $ spire https export homeworld.mit.edu ./homeworld.mit.edu.key ./homeworld.mit.edu.pem

# Configuring SSH

Configure SSH so that it has the correct certificate authority in ~/.ssh/known_hosts for members of the cluster:

    $ spire access update-known-hosts

# Building the ISO

Now, create an ISO:

    $ spire iso gen preseeded.iso ~/.ssh/id_rsa.pub   # this SSH key is used for direct access during cluster setup

Now you should burn and/or upload preseeded.iso that you've just gotten, so that you can use it for installing servers. Make a note of the password it generated.

For development on the official homeworld development servers (the LocalForward lines set up port forwarding for VNC):

    $ edit ~/.ssh/config
        Host toast
                HostName toastfs-dev.mit.edu
                User root
                GSSAPIAuthentication yes
                GSSAPIKeyExchange no
                GSSAPIDelegateCredentials no
                LocalForward 5901 localhost:5901
                LocalForward 5902 localhost:5902
                LocalForward 5903 localhost:5903
                LocalForward 5904 localhost:5904
                LocalForward 5905 localhost:5905
                LocalForward 5906 localhost:5906
                LocalForward 5910 localhost:5910
    $ scp preseeded.iso toast:/srv/preseeded.iso
        # If you get a password prompt, DON'T give it anything.
        # You should generate Kerberos tickets (kinit) before
        # attempting to access the development server, otherwise
        # SSH will fallback to password authentication.

# Setting up the machines

## For development only: Rebuilding the virtual machines

For development, we're using a set of virtual machines on toast. To simulate cluster bringup, we destroy all the virtual machines and rebuild them using a script on toast. On toast (access toast with ``ssh toast``):

    $ ~/hyades/rebuild-homeworld-cluster.sh /srv/preseeded.iso

You can then access the virtual machines using VNC. For example, using TigerVNC:

    $ vncviewer localhost:5910 # supervisor node
    $ for i in `seq 1 6`; do vncviewer localhost:590$i & done

## Setting up the supervisor operating system

 * Boot the ISO on the hardware
   - Select `Install`
   - Enter the IP address for the server (18.181.0.253 on our test infrastructure)
   - Wait a while
   - Enter "manual" for the bootstrap token (so that your SSH keys will work)
 * Log into the server directly with your SSH keys
   - Verify the host keys based on the text printed before the login console

For example, ``ssh root@egg-sandwich.mit.edu``. You might need to remove previous SSH hostkeys from known_hosts if you've set up the cluster before.

## Setting up ssh-agent

If you don't already have ssh-agent running:

    $ eval `ssh-agent -s`
    $ ssh-add

Note that this is a local change that does not persist on reboot.

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

For development, currently the order in which the nodes are listed is deceiving. This should be fixed soon, but here's a reference for now.

   - master01: eggs-benedict
   - master02: huevos-rancheros
   - master03: ole-miss
   - worker01: grilled-cheese
   - worker02: avocado-burger
   - worker03: french-toast

Boot the ISO on each piece of hardware
   - Select `Install`
   - Enter the IP address for the server
   - Wait a while
   - Enter the bootstrap token

Confirm that all of the servers came up properly (and requested their keys
correctly):

    $ spire verify online
      # if this fails, it's possible that your ssh-agent might be borked and you need to restart it.

# Core cluster bringup

Bring up the core cluster:

    $ spire seq core

If and only if you're hosting the containers for core cluster services on the cluster itself:

    $ spire seq registry

Deploy flannel into the cluster:

    $ spire seq flannel

Deploy dns-addon into the clustesr:

    $ spire seq dns-addon

# Finishing up

The cluster should now be ready!
