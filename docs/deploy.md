# Deployment

There are a couple of different ways to do the various deployment steps,
so the docs are broken up into a few different files.
The general skeleton is:

- Obtain a Debian buster environment.
  You may use rhombi.mit.edu for chroot installation.
- Prepare a deployment environment: any of
  - [Direct installation](installation-direct.md)
  - [Installation in a chroot](installation-chroot.md)
- Deploy a cluster: any of
  - [Setting up a brand new cluster](cluster-new.md)
  - [Redeploying an existing cluster](cluster-redeploy.md)
  - [Deploying a transient virtual cluster](cluster-autodeploy.md)
