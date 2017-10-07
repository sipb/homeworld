# How to deploy a Homeworld cluster

## Installing packages

You will need to install homeworld-admin-tools and all its dependencies. This
will provide access to the 'spire' tool.

TODO: instructions on setting this up.

## Setting up a workspace

You need to a set up an environment variable corresponding to a folder that can
store your cluster's configuration and authorities.

    $ export HOMEWORLD_DIR="$HOME/my-cluster"
    $ spire config populate
    $ spire config edit

## Generate authority keys

    $ spire authority gen

## Building the ISO

Now, create an ISO:

    $ spire iso gen preseeded.iso ~/.ssh/id_rsa.pub   # this key is used for direct access during cluster setup

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

## Set up the keyserver

 * Request a keytab from accounts@, if necessary
 * Rotate the keytab (and upgrade its cryptographic strength):

       $ k5srvutil -f <keytab> change -e aes256-cts:normal,aes128-cts:normal
         # the following will invalidate current tickets:
       $ k5srvutil -f <keytab> delold
       $ cp <keytab> $HOMEWORLD_DIR/keytab.<hostname>   # i.e. keytab.egg-sandwich

 * Configure the supervisor keyserver:

       $ spire setup keyserver
       $ spire verify keystatics   # make sure the keyserver is running

 * Admit the supervisor node to the cluster:

       $ spire setup self-admit

 * Prepare kerberos gateway:

       $ spire setup keygateway
       $ spire verify keygateway

## Request certificates and SSH with them

 * Request SSH cert:

       $ spire access update-known-hosts    # set up certificate authority in ~/.ssh/known_hosts
       $ spire access ssh    # if this fails, you might need to make sure you don't have any stale kerberos tickets

 * Configure and test SSH:

       $ # this will deny your current direct access, so keep a SSH session open until you verify this works
       $ spire setup supervisor-ssh
       $ spire verify ssh-with-certs
       $ # if that worked, you can close your other SSH session

## Set up each node's operating system

 * Request a bootstrap token:

       $ spire infra admit <hostname>.mit.edu
       Token granted for <hostname>.mit.edu: '<TOKEN>'

 * Boot the ISO on the hardware
   - Select `Install`
   - Enter the IP address for the server (18.181.X.Y on our test infrastructure)
   - Wait a while
   - Enter the bootstrap token
 * Confirm that the server came up properly (and requested its keys correctly):

        $ spire verify online <hostname>      # you might need to re-request certificates first

## Package installation

 * Install and upgrade packages on all systems:

        $ spire infra install-packages

## Core cluster bringup

 * Launch services

        $ spire setup services
        $ spire verify etcd        # cursory inspection of etcd
        $ spire verify kubernetes  # cursory inspection of kubernetes

## Bootstrap cluster registry

This step is needed when you're hosting the containers for core cluster
services on the cluster itself.

    $ mkdir $HOMEWORLD_DIR/https-certs
    $ cp homeworld.mit.edu.key homeworld.mit.edu.pem $HOMEWORLD_DIR/https-certs
    $ spire setup dns-bootstrap
    $ spire setup bootstrap-registry
    $ spire verify aci-pull

## Core cluster service: flannel

Deploy flannel into the cluster:

    $ mkdir cluster-gen
    $ spire config gen-kube cluster-gen
    $ spire kubectl create -f cluster-gen/flannel.yaml

Wait a bit for propagation... (if this doesn't work, keep trying for a bit)

    $ spire verify flannel-run
    $ spire verify flannel-listen

## Core cluster service: dns-addon

Deploy dns-addon into the clustesr:

    $ spire kubectl create -f dns-addon.yaml

Wait a bit for propagation... (if this doesn't work, keep trying for a bit)

    $ spire verify dns-addon-run
    $ spire verify dns-addon-query

## Finishing up

The cluster should now be ready!
