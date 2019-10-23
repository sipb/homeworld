# Setting up a new cluster from scratch

**NOTE**: Some of the information in this document is specific to the Hyades project.

**NOTE**: If you're re-deploying the cluster for development,
follow [Deploying a prepared cluster](cluster-redeploy.md) instead.
You might additionally want to regenerate authority keys (`spire authority gen`) --
but you'd need to push to hyades-cluster if you make any changes to the cluster configuration.

## Setting up a workspace

You need to a set up an environment variable corresponding to a folder
that can store your cluster's configuration and authorities.
The hyades workspace is in a private git repository called hyades-cluster.
(Assuming that your disaster recovery key (see below) is well-protected,
this folder can be publicly-readable, but this is discouraged.)
Make very certain that you only commit encrypted keys.

    $ export HOMEWORLD_DIR="$HOME/my-cluster"

## Setting up secure key storage

You need to choose a location to hold the disaster recovery key for your cluster.
If your cluster is for development purposes, it will suffice to store it locally,
but for production clusters it should be stored offline on something like an encrypted USB drive.

    $ export HOMEWORLD_DISASTER="/media/usb-crypt/homeworld-disaster"

This key will be used to encrypt the private authority keys.

**WARNING**: because gpg's `--passphrase-file` option is used,
only the first line from the file will be used as the key!

**WARNING**: The disaster recovery key is used to encrypt upstream keys.
If you are rotating the disaster recovery key, you should first decrypt the upstream keys:

    $ spire keytab export egg-sandwich egg-keytab
    $ spire https export homeworld.mit.edu ./homeworld.key ./homeworld.pem

Recommended method of generating the passphrase:

    $ pwgen -s 160 1 >$HOMEWORLD_DISASTER

Make sure that you do not do this on a multi-user system,
or that you've otherwise protected the file that you're writing out from others.

## Configuring the cluster

Set up the configuration:

    $ spire config populate
    $ spire config edit

## Generating authority keys

    $ spire authority gen

## Acquiring upstream keys

 * Request a keytab from accounts@, if necessary
 * Import the keytab into the project:

```
$ spire keytab import <hostname> <path-to-keytab>
```

 * Rotate the keytab (which includes upgrading its cryptographic strength):

```
$ spire keytab rotate <hostname>
   # the following means invalidating current tickets:
$ spire keytab delold <hostname>
```

 * If you are using the user grant system, import the HTTPS key and certificate for the host you configured:

```
$ spire https import homeworld.mit.edu ./homeworld.key ./homeworld.pem
```

Now you can consider putting this folder in Git, and then move on to [deploying the new cluster](cluster-redeploy.md).

## Uploading to Git

Be very careful not to add unencrypted key material, because that could cause a security breach.

    $ cd $HOMEWORLD_DIR
    $ git init
    $ git add setup.yaml authorities.tgz keytab.*.crypt https.*    # be VERY CAREFUL about what you're adding!
    $ git commit
    $ git remote add origin ...
    $ git push -u origin master
