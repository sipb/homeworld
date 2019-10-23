# Installation in a chroot

You will need a Debian stretch installation or VM --
note that we do not support any other environments.
Hyades project members can use the rhombi.mit.edu machine for this.

From your build chroot, run:

    [homeworld] $ platform/extract-apt-setup.sh

If your build and deploy environments are in different locations,
then copy `apt-setup.deb` to the homeworld directory in your deploy environment.
Now run:

    $ export HOMEWORLD_DEPLOY_CHROOT="/path/to/deploy/chroot"
    $ deploy-chroot/create.sh  # create a deploy chroot
    $ deploy-chroot/enter.sh   # enter the chroot

    [hyades deploy] $ reinstall-apt-setup.sh
    [hyades deploy] $ spire version

Rerun `extract-apt-setup.sh` and `reinstall-apt-setup.sh`
whenever you make changes in homeworld that you want to test.
You can also use `reinstall-spire.sh` if spire was the only thing you changed.

The following environment paths are automatically mounted in the deployment chroot;
set them before calling `enter.sh`:
- `HOMEWORLD_DIR`
- `HOMEWORLD_DISASTER`
