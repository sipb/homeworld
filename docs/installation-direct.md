# Direct installation on stretch

You will need a Debian stretch installation or VM --
note that we do not support any other environments.

Start by getting the apt-setup package.  Inside your build chroot, run:

    [homeworld] $ tools/extract-apt-setup.sh

This places `apt-setup.deb` in your host homeworld directory.
Copy `apt-setup.deb` to your stretch installation and run:

    $ sudo dpkg -i apt-setup.deb
    $ sudo apt-get update
    $ sudo apt-get install homeworld-spire

Check that spire is working properly:

    $ spire version
