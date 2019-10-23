# Autodeploy

This is a lightweight, automated way of deploying a virtualized cluster for testing.

## Configuration

To get an autodeploy configuration:

    $ cp deploy-chroot/setup.yaml.in $HOMEWORLD_DIR
    $ deploy-chroot/generate-setup.py <x>

Here `<x>` is a byte which must be unique per machine
if you plan to do multiple concurrent autodeploys.
If you are working on rhombi, please coordinate assignment of these numbers
via the `/lock` bot on mattermost.

## Autodeploy

    $ eval `ssh-agent -s` && ssh-add  # if not already running
    $ spire virt auto install

That's it.
