# How to deploy a Homeworld cluster

## Basic builds

To build a new ISO, although you don't need everything built, you do need three
packages built:

 * homeworld-apt-setup
 * homeworld-knc
 * homeworld-keysystem
 * homeworld-admin-tools

See [build.md](build.md) for details.

## Installing packages

You will need to install homeworld-admin-tools and all its dependencies. This
will provide access to the 'spire' tool.

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
       $ ssh -v root@<hostname>.mit.edu
         # ensure that a debug line like this shows up:
         debug1: Server accepts key: pkalg ssh-rsa-cert-v01@openssh.com blen 1524
         # (if there's no ssh-rsa-cert-v01, certs might not be set up properly)
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

## Confirm etcd works

 * Query etcd cluster health:

        $ spire etcdctl cluster-health
        rotating etcd certs...
        member 439721bf885a52a5 is healthy: got healthy result from https://18.181.0.104:2379
        member 61712dffdce48432 is healthy: got healthy result from https://18.181.0.97:2379
        member f6d798ec325cf15d is healthy: got healthy result from https://18.181.0.106:2379
        cluster is healthy

 * Query etcd cluster members:

        $ spire etcdctl member list
        439721bf885a52a5: name=huevos-rancheros peerURLs=https://18.181.0.104:2380 clientURLs=https://18.181.0.104:2379 isLeader=false
        61712dffdce48432: name=eggs-benedict peerURLs=https://18.181.0.97:2380 clientURLs=https://18.181.0.97:2379 isLeader=true
        f6d798ec325cf15d: name=ole-miss peerURLs=https://18.181.0.106:2380 clientURLs=https://18.181.0.106:2379 isLeader=false

## Confirm kubernetes works

 * Query default cluster setup:

        $ spire kubectl get nodes
        NAME               STATUS                     AGE       VERSION
        avocado-burger     Ready                      16m       v1.7.2+$Format:%h$
        eggs-benedict      Ready,SchedulingDisabled   16m       v1.7.2+$Format:%h$
        french-toast       Ready                      16m       v1.7.2+$Format:%h$
        grilled-cheese     Ready                      16m       v1.7.2+$Format:%h$
        huevos-rancheros   Ready,SchedulingDisabled   16m       v1.7.2+$Format:%h$
        ole-miss           Ready,SchedulingDisabled   16m       v1.7.2+$Format:%h$
        $ spire kubectl get namespaces
        NAME          STATUS    AGE
        default       Active    17m
        kube-public   Active    17m
        kube-system   Active    17m
        $ spire kubectl get all --namespace=default
        NAME             CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
        svc/kubernetes   172.28.0.1   <none>        443/TCP   17m
        $ spire kubectl get all --namespace=kube-public
        No resources found.
        $ spire kubectl get all --namespace=kube-system
        No resources found.

## Bootstrap cluster DNS

This step is needed when you're hosting the containers for core cluster
services on the cluster itself.

    $ spire setup dns-bootstrap

We don't yet have the system to a point where you can stop needing to bootstrap
DNS, but when that happens, you can turn it back off:

    $ spire setup stop-dns-bootstrap

## Bootstrap cluster registry

    $ mkdir $HOMEWORLD_DIR/https-certs
    $ cp homeworld.mit.edu.key homeworld.mit.edu.pem $HOMEWORLD_DIR/https-certs
    $ spire setup bootstrap-registry

## Confirm container launching

    $ ssh root@<worker-hostname>.mit.edu
    # rkt run --debug --interactive=true homeworld.mit.edu/debian
        $ ping 8.8.8.8
        $ exit

## Core cluster service: flannel

Deploy flannel into the cluster:

    $ mkdir cluster-gen
    $ spire config gen-kube cluster-gen
    $ spire kubectl create -f cluster-gen/flannel.yaml

Wait a bit for propagation.

    $ spire kubectl get pods --namespace=kube-system
    NAME                    READY     STATUS    RESTARTS   AGE
    kube-flannel-ds-1r1cx   1/1       Running   0          49s
    kube-flannel-ds-2cxj5   1/1       Running   0          49s
    kube-flannel-ds-33rfs   1/1       Running   0          49s
    kube-flannel-ds-533p8   1/1       Running   0          49s
    kube-flannel-ds-9sw4x   1/1       Running   0          49s
    kube-flannel-ds-k52q1   1/1       Running   0          49s

Verify flannel functionality by running flannel tests on two different nodes:

    $ # on two different nodes
    $ ssh root@<worker>.mit.edu
    # rkt run --debug --interactive=true --net=rkt.kubernetes.io homeworld.mit.edu/debian
        $ ip addr   # make sure this provides a 172.18 IP, and not a 172.16 IP.
        $ ping <other-172.18-addr>

If the ping works both ways, then flannel works! At least at a basic level.

## Core cluster service: dns-addon

Deploy dns-addon into the cluster:

    $ spire kubectl create -f dns-addon.yaml

Wait for deployment to succeed:

    $ spire kubectl get pods --namespace=kube-system
    NAME                    READY     STATUS    RESTARTS   AGE
    kube-dns-v20-69lrg      3/3       Running   0          1m
    kube-dns-v20-clh2z      3/3       Running   0          1m
    kube-dns-v20-fpvf9      3/3       Running   0          1m

Verify that DNS works:

    $ ssh root@<worker>.mit.edu
    # apt-get install dnsutils
    # nslookup kubernetes.default.svc.hyades.local 172.28.0.2
    Address: 172.28.0.1
    # rkt run --debug --interactive=true --net=rkt.kubernetes.io homeworld.mit.edu/debian
        $ nslookup kubernetes.default.svc.hyades.local 172.28.0.2
        Address: 172.28.0.1

## Finishing up

Now the cluster is prepared! It sounds like a good time to help develop the
cluster code further.
