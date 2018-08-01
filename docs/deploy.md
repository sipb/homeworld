# Prerequisites

If you're re-deploying the cluster for development, you will need:

 * A Debian Stretch installation (or VM) -- note that we do not support any other environments.
 * The disaster recovery key.
 * Access to [hyades-cluster](https://github.mit.edu/sipb/hyades-cluster), where we store the current cluster configuration. You will need to have set up SSH keys with github.mit.edu.
 * Your Kerberos identity (preferably a [root instance](https://sipb.mit.edu/doc/root-instance/)) in the root-admins secion of ``setup.yaml``. If it isn't there, you can just add it in yourself.
 * Access to toastfs-dev (the machine which hosts the development cluster). You will need a Kerberos root instance as a prerequisite to this.
 * Any VNC viewer. These instructions are based on [TigerVNC](https://github.com/TigerVNC/tigervnc/releases) (``sudo apt-get install tigervnc``).

# Installing packages

Copy building/binaries/<domain>/<subbranch>/homeworld-apt-setup_0.1.5_amd64.deb directly to the target VM from your build machine.

TODO: instructions for the case where this transfer is not easy.

Then, in your deploy VM:

    $ sudo dpkg -i homeworld-apt-setup_0.1.5_amd64.deb
    $ sudo apt-get update
    $ sudo apt-get install homeworld-admin-tools

This will provide access to the 'spire' tool.

# Setting up a new cluster from scratch

**NOTE**: If you're re-deploying the cluster for development, follow [Deploying a prepared cluster](#deploying-a-prepared-cluster) instead. You might additionally want to regenerate authority keys (``spire authority gen``) -- but you'd need to push to hyades-cluster if you make any changes to the cluster configuration.

## Setting up a workspace

You need to a set up an environment variable corresponding to a folder that can store your cluster's configuration and authorities. Assuming that your disaster recovery key (see below) is well-protected, this folder can be a publicly-readable git repository. Make very certain that you only commit encrypted keys.

    $ export HOMEWORLD_DIR="$HOME/my-cluster"

## Setting up secure key storage

You need to choose a location to hold the disaster recovery key for your cluster. If your cluster is for development purposes, it will suffice to store it locally, but for production clusters it should be stored offline on something like an encrypted USB drive.

    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"

This key will be used to encrypt the private authority keys.

**WARNING**: because gpg's `--passphrase-file` option is used, only the first line from the file will be used as the key!

**WARNING**: The disaster recovery key is used to encrypt upstream keys. If you are rotating the disaster recovery key, you should first decrypt the upstream keytab:

    $ spire keytab export egg-sandwich egg-keytab

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

**WARNING**: If you're re-deploying the cluster for development, you should not be following this section (unless you are encrypting upstream keys with a newly generated disaster recovery key). Critically, you should not rotate the keytab, or you'd need to distribute the new keytab to everyone. Follow [Deploying a prepared cluster](#deploying-a-prepared-cluster) instead.

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

Now you can consider putting this folder in Git, and then move on to 'Deploying a prepared cluster' below.

## Uploading to Git

Be very careful not to add unencrypted key material, because that could cause a security breach.

    $ cd $HOMEWORLD_DIR
    $ git init
    $ git add setup.yaml authorities.tgz keytab.*.crypt    # be VERY CAREFUL about what you're adding!
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

# Configuring SSH

Configure SSH so that it has the correct certificate authority in ~/.ssh/known_hosts for members of the cluster:

    $ spire access update-known-hosts

# Building the ISO

Now, create an ISO:

    $ spire iso gen preseeded.iso ~/.ssh/id_rsa.pub   # this SSH key is used for direct access during cluster setup

Now you should burn and/or upload preseeded.iso that you've just gotten, so that you can use it for installing servers.

For development on the official homeworld servers (the LocalForward lines set up port forwarding for VNC):

    $ edit ~/.ssh/config
        Host toast
                HostName toastfs-dev.mit.edu
                User root
                GSSAPIAuthentication yes
                GSSAPIKeyExchange no
                GSSAPIDelegateCredentials no
                PreferredAuthentications gssapi-with-mic
                LocalForward 5901 localhost:5901
                LocalForward 5902 localhost:5902
                LocalForward 5903 localhost:5903
                LocalForward 5904 localhost:5904
                LocalForward 5905 localhost:5905
                LocalForward 5906 localhost:5906
                LocalForward 5910 localhost:5910
        Host toast-noforward
                HostName toastfs-dev.mit.edu
                User root
                GSSAPIAuthentication yes
                GSSAPIKeyExchange no
                GSSAPIDelegateCredentials no
                PreferredAuthentications gssapi-with-mic

        # Note that you will need Kerberos tickets.
        # Generate them for your Kerberos identity from the root-admins section of setup.yaml, via
        # kinit <kerberos principal>
        # to access the development server.
    $ scp preseeded.iso toast-noforward:/srv/preseeded.iso

# Setting up the machines

## For development only: Rebuilding the virtual machines

For development, we're using a set of virtual machines on toast. To simulate cluster bringup, we destroy all the virtual machines and rebuild them using a script on toastfs-dev. On toastfs-dev (access with ``ssh toast``):

    # hyades/rebuild-homeworld-cluster.sh /srv/preseeded.iso

You can then access the virtual machines using VNC. For example, using TigerVNC:

    $ vncviewer localhost:5910 # supervisor node
    $ for i in `seq 1 6`; do vncviewer localhost:590$i 2>/dev/null & done

Note that you will need a toastfs-dev SSH session running so that VNC can communicate through it (via LocalForward).

## Setting up the supervisor operating system

 * Boot the ISO on the hardware
   - Select `Install`
   - Enter the IP address for the server (18.4.60.150 on our test infrastructure)
   - Wait a while
   - Enter "manual" for the bootstrap token (so that your SSH keys will work)
 * Log into the server directly with your SSH keys
   - For example, ``ssh root@egg-sandwich.mit.edu true``. You might need to remove previous SSH host keys from known_hosts if you've set up the cluster before.
   - Verify the host keys based on the text printed before the login console

## Setting up ssh-agent

If you don't already have ssh-agent running:

    $ eval `ssh-agent -s`
    $ ssh-add

Note that this is a local change that does not persist on reboot.

## Setting up the supervisor node

Set up the keysystem, SSH, and prometheus:

    $ spire seq supervisor

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

Boot the ISO on each piece of hardware:

   - Select `Install`
   - Enter the IP address for the server
   - Wait a while
   - Enter the bootstrap token

Finally, run the verification script to watch the cluster come online:

    $ spire seq cluster

Note that this script doesn't perform any actions; it simply tracks state.

If it fails, the cluster did not come up correctly.
It might either take more time (try running the command again),
or there's a problem preventing the cluster from becoming properly configured.

# Finishing up

The cluster should now be ready to work with!
