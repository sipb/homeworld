# How to deploy a Homeworld cluster

## Building Software

 * Install sbuild and dependencies from debian repositories.
 * Build helpers:
   - In `packages/helper-go`, run `./build.sh`.
   - In `packages/helper-acbuild`, run `./build.sh`.
 * Build packages: for each folder named `packages/homeworld-*`, go into it and
   `./build-packages`. (You may need to allocate additional memory to your
   build environment for building kubernetes.)
 * Rebuild and/or update the repository by going into `repository` and running
   `./rebuild.sh`.

## Provisioning a server

 * Generate a preseeded ISO by going into `installation` and running
   `generate.sh`. You will need a few packages, such as cpio and genisoimage.
   - Keep track of the password that it outputted.
 * Boot the ISO on the target system.
   - Select `Install`
   - Enter the last two octets of the IP address for the server.
   - Installation should be entirely automatic besides these two steps.
 * Log into the server directly with the aforementioned root password.

## Set up the authentication server

 * Request a keytab from accounts@, if necessary.
 * Provision a server as above; set up direct SSH key access for now.
 * Generate ssh user CA locally; save it somewhere safe.
 * Rotate the keytab locally (k5srvutil -f <keytab> change); save it somewhere safe.
 * Run `auth/deploy.sh` on `<host> <keytab> auth-login <user-ca>`
 * Run `req-cert` and see if it works.
 * Confirm that you can log into the server with kerberos auth.
 * Remove your direct SSH key access.

## Initial server setup

 * Provision a server as above.
 * Launch an admission server from a trusted machine. Copy up the relevant files.
 * Admit the server according to the instructions. Verify all hashes carefully.
 * Make sure to add the CA key for the server into your known_hosts.

       @cert-authority eggs-benedict.mit.edu,huevos-rancheros.mit.edu,[...] ssh-rsa ...

 * Confirm that you can ssh into the server as root with certs gotten from `req-cert`.

## Configuration and SSL setup and package installation

 * Modify config/setup.conf
 * Run ./compile-config.py
 * Run ./compile-certificates cluster-config/certificates.list <secrets-directory>
 * Run pkg-install-all.sh
 * If this is the first time installing this cluster, run authority-gen.sh
 * Run certify.sh
 * Run spin-up-all.sh

## Starting everything

 * Manually start etcd on each master node
 * Run init-flannel.sh on one master node
 * Run start-master.sh on all master nodes
 * Run start-worker.sh on all worker nodes

 * Run kubectl and make sure things work (you may need to generate certs for this)

## Core cluster services

 * kubectl create -f dns-addon.yml
