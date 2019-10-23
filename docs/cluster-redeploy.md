# Deploying a prepared cluster

For deploying to the Hyades development cluster, you will need:

- Access to [hyades-cluster](https://github.mit.edu/sipb/hyades-cluster),
  where we store the current cluster configuration.
  You will need to have set up SSH keys with github.mit.edu.
- Your Kerberos identity (preferably a [root instance](https://sipb.mit.edu/doc/root-instance/)) in the root-admins section of ``setup.yaml``.
  If it isn't there, you can just add it in yourself.
- Access to toastfs-dev (the machine which hosts the development cluster).

## Cloning an existing cluster configuration

To download existing configuration:

    $ export HOMEWORLD_DIR="$HOME/my-cluster"
    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"
    $ git clone git@github.mit.edu:sipb/hyades-cluster $HOMEWORLD_DIR

Make sure to verify that you have the correct commit hash, out of band.

## Configuring SSH

Configure SSH so that it has the correct certificate authority in ~/.ssh/known_hosts for members of the cluster:

    $ spire access update-known-hosts

## Building the ISO

    $ spire iso gen preseeded.iso ~/.ssh/id_rsa.pub   # this SSH key is used for direct access during cluster setup

Now you should burn and/or upload `preseeded.iso`
so that you can use it for installing servers.

## For development only: Rebuilding the virtual machines

For development, we're using a set of virtual machines on toastfs-dev.
The following SSH config can be used for accessing that machine:

    $ edit ~/.ssh/config
        Host toast-vnc
                HostName toastfs-dev.mit.edu
                User root
                GSSAPIAuthentication yes
                GSSAPIKeyExchange no
                GSSAPIDelegateCredentials no
                PreferredAuthentications gssapi-with-mic
                # Port forwarding for VNC
                LocalForward 5901 localhost:5901
                LocalForward 5902 localhost:5902
                LocalForward 5903 localhost:5903
                LocalForward 5904 localhost:5904
                LocalForward 5905 localhost:5905
                LocalForward 5906 localhost:5906
                LocalForward 5910 localhost:5910
        Host toast
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

To simulate cluster bringup, we destroy all the virtual machines and rebuild them using a script on toastfs-dev.

    $ scp preseeded.iso toast:/srv/preseeded.iso
    $ ssh toast
    toastfs-dev:# hyades/rebuild-homeworld-cluster.sh /srv/preseeded.iso

You can then access the virtual machines using VNC.
For example, using TigerVNC:

    $ sudo apt-get install tigervnc
    $ ssh -n toast-vnc 'sleep infinity' &  # leave this running in the background
    $ vncviewer localhost:5910 # supervisor node
    $ for i in `seq 1 6`; do vncviewer localhost:590$i 2>/dev/null & done

Note that you need to leave the toast-vnc SSH session running
so that VNC can communicate through it.

## Setting up the supervisor operating system

- Boot the ISO on the hardware
  - Select `Install`
  - Enter the IP address for the server (see `setup.yaml`)
  - Wait a while
  - Enter "manual" for the bootstrap token (so that your SSH keys will work)
- Log into the server directly with your SSH keys
  - For example, ``ssh root@egg-sandwich.mit.edu true``.
    You might need to remove previous SSH host keys from known_hosts
    if you've set up the cluster before.
  - Verify the host keys based on the text printed before the login console

## Setting up ssh-agent

If you don't already have ssh-agent running:

    $ eval `ssh-agent -s`
    $ ssh-add

Note that this is a local change that only affects your current session.

## Setting up the supervisor node

Set up the keysystem, SSH, and prometheus:

    $ spire seq supervisor

## Set up each node's operating system

Request bootstrap tokens:

    $ spire infra admit-all

The VMs on toastfs-dev correspond to hostnames as follows:

- supervisor: egg-sandwich
- master01: eggs-benedict
- master02: huevos-rancheros
- master03: ole-miss
- worker01: grilled-cheese
- worker02: avocado-burger
- worker03: french-toast

Boot the ISO on each piece of hardware:

   - Select `Install`
   - Enter the IP address for the server (see `setup.yaml`)
   - Wait a while
   - Enter the bootstrap token

Finally, run the verification script to watch the cluster come online:

    $ spire seq cluster

Note that this script doesn't perform any actions; it simply tracks state.
If it fails, the cluster did not come up correctly.
It might either take more time (try running the command again),
or there's a problem preventing the cluster from becoming properly configured.
