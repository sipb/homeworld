# How to deploy a Homeworld cluster

## Building Software

 * Install go, acbuild, and ruby. See trust.txt for signatures.
 * Install fpm from the gems in build-rkt.
 * Build hyauth, etcd, and flannel with the ./build.sh scripts.
 * Make sure you have these installed:
    git build-essential zlib1g-dev libxml2 libxml2-dev libreadline-dev
    libssl-dev zlibc automake squashfs-tools libacl1-dev libsystemd-dev
    libcap-dev libglib2.0-dev libpcre3-dev libpcrecpp0 libpixman-1-dev
    pkg-config realpath flex bison
 * For rkt, run ./build.sh.
    You may need to tweak the removal of stage1/usr_from_kvm/kernel/patches/0002-for-debian-gcc.patch if you get '-no-pie' problems.
 * For kubernetes, run ./build.sh
    You may need to allocate additional memory to your build environment.
 * ./build-package.sh on deployment

## Set up the authentication server

(IF YOU ARE REINITIALIZING AN AUTHSERVER, MAKE SURE TO COPY OFF THE KEYTAB AND SSH CA!)

TODO: automate this process better

 * Request a keytab from accounts@, if necessary.
 * Provision yourself a Debian Stretch machine. Choose SSH Server and Standard System Utilities.
   * Remove irrelevant things like exim4 if necessary.
 * Install krb5-user
 * Stick keytab into /etc/krb5.keytab
 * k5srvutil change
 * Consider backing up new keytab
 * Create user kauth with a homedir
 * Set up the ACL for the authserver in kauth's .k5login.
 * Generate yourself a SSH user CA, if necessary. '/home/kauth/ca_key'.
 * Back up the user CA somewhere, probably?
 * Copy the user CA pubkey off of the server.
 * Setup SSH access via kerberos, or, if you don't have the keytab yet, SSH keys.
 * Turn off PasswordAuthentication.
 * Set up a .k5login in the homedir of any user with shell access, even if empty.
 * Upload hyauth to /usr/local/bin/hyauth
 * Set hyauth as kauth's shell

 * Run req-cert and see if it works.

TODO: improve cryptographic strength of keytab (see note on sipb page)

## Initial server setup

 * Provision yourself new Debian Stretch machines. Choose SSH Server only.
 * Launch an admission server from a trusted machine. Copy up the relevant files.
 * Admit the server according to the instructions. Verify all hashes carefully.
 * Make sure to add the CA key for the server into your known_hosts.

       @cert-authority eggs-benedict.mit.edu,huevos-rancheros.mit.edu,[...] ssh-rsa ...

 * Confirm that you can ssh into the server as root.

## Configuration and SSL setup and package installation

 * Install openssl, curl, and ca-certificates on each server
 * Modify config/setup.conf
 * Run ./compile-config.py
 * Run ./compile-certificates cluster-config/certificates.list <secrets-directory>
 * Run authority-gen.sh
 * Run authority-upload.sh
 * Run private-gen.sh
 * Run shared-gen.sh
 * Run shared-upload.sh
 * Run certificate-gen-csrs.sh
 * Run certificate-sign-csrs.sh
 * Run certificate-upload-certs.sh
 * Run spin-up-all.sh
 * Run pkg-install-all.sh on the latest packages and etcd-current-linux-amd64.aci

## Starting everything

 * Manually start etcd on each master node
 * Run init-flannel.sh on one master node
 * Run start-master.sh on all master nodes
 * Run start-worker.sh on all worker nodes

 * Run kubectl and make sure things work (you may need to generate certs for this)

## Core cluster services

 * kubectl create -f dns-addon.yml
