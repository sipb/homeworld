# How to deploy a Homeworld cluster

## Generating admission keys

To generate infrastructural access keys:

    $ cd deployment/admit
    $ ./generate-secrets ??/secrets/ <AUTHSERVER>.mit.edu

You'll now have the certificate authorities for the admission process and the
certificate authorities for ongoing cluster access.

## Building the ISO

You will need to install, at the very least:

 * build-essential
 * sbuild
 * cpio
 * genisoimage

And probably some other miscellaneous things too.

To build a new ISO, although you don't need everything built, you do need two
packages built:

 * homeworld-apt-setup
 * homeworld-admitclient

To do so, you need to build the go compiler first. So:

    $ cd building/build-helpers/helper-go
    $ ./build.sh

    $ cd building/build-debs/homeworld-apt-setup
    $ ./build-package.sh

    $ cd building/build-debs/homeworld-admitclient
    $ ./build-package.sh

    $ cd building/build-iso
    $ ./generate.sh <AUTHSERVER>.mit.edu ??/secrets/admission.pem

Now you should burn and/or upload the .iso that you've just gotten, so that you
can use it for installing servers. Make a note of the password it generated.

## Provisioning a server

 * Boot the ISO on the target system.
   - Select `Install`
   - Enter the last two octets of the IP address for the server.
   - Installation should be entirely automatic besides these two steps.
 * Log into the server directly with the aforementioned root password.

## Set up the authentication server

 * Request a keytab from accounts@, if necessary.
 * Provision a server as above; set up direct SSH key access for now.
 * Until you've verified that kerberos auth works (below), keep a SSH session
   open continously, just in case.
 * Rotate the keytab (and upgrade its cryptographic strength):

       $ k5srvutil -f <keytab> change -e aes256-cts:normal,aes128-cts:normal
         # the following will invalidate current tickets:
       $ k5srvutil -f <keytab> delold
       $ cp <keytab> <secret-dir>/
       $ scp <keytab> root@HOSTNAME.mit.edu:/etc/krb5.keytab

 * Run `auth/deploy.sh` on `<host> <keytab> auth-login <user-ca>`
 * Make sure you don't have any stale tickets
 * Run `req-cert` and see if it works.
 * Confirm that you can log into the server with kerberos auth.
 * Remove your direct SSH key access.

## Set up the admission server

This probably goes on the same box as the authentication server.

    $ cd deployment-config
    $ nano setup.conf
    $ ./compile-config.py

    $ cd admit
    $ ./deploy.sh AUTHSERVER.mit.edu ???/secrets ../deployment-config/cluster-config/

The deployment should finish successfully.

## Initial node setup

 * Provision a server as above.
 * Locally, run `$ admit/prepare-admit.sh ???/secrets AUTHSERVER.mit.edu new-node-hostname`
     (The hostname should not contain the .mit.edu.)
 * Run `# pull-admit <TOKEN>` on the new server, with the token produced by prepare-admit.sh.
 * Make sure to add the CA key for the server into your known_hosts.

       @cert-authority eggs-benedict.mit.edu,huevos-rancheros.mit.edu,[...] ssh-rsa ...

 * Confirm that you can ssh into the server as root with certs gotten from `req-cert`.

## Configuration and SSL setup and package installation

 * Run deployment-config/compile-certificates cluster-config/certificates.list <secrets-directory>
 * Run pkg-install-all.sh
 * If this is the first time installing this cluster, run authority-gen.sh
 * Run certify.sh

## Starting everything

 * Run start-all.sh as generated during the configuration phase
 * Run etcdctl and make sure things work (you may need to generate certs for this)
 * Run kubectl and make sure things work (you may need to generate certs for this)

## Core cluster services

 * Go into clustered/
 * Generate flannel config: generate.sh ../deployment-config/cluster-config/cluster.conf
 * Deploy: kubectl create -f flannel.yml
 * Verify flannel functionality by using two homeworld.mit.edu/debian containers.
 * Set up DNS: kubectl create -f dns-addon.yml
 * Verify DNS: nslookup kubernetes.default.svc.hyades.local 172.28.0.2
     "Address: 172.28.0.1"
