# Hyades Provisioning

Etherpad: http://etherpad.mit.edu/p/hyades-provisioning-todo

AFS: /afs/sipb/project/hyades/provisioning

# Subfolders

 * auth: authserver
 * admission: scripts to set up the SSH system
 * build-etcd, build-flannel, build-kubernetes, build-rkt: building required software components
 * deployment: deployment code and overview
 * clustered: code that runs on the cluster

# Repository Security

All commits in this repository are signed with GPG:

    pub   rsa4096 2016-10-12 [SC] [expires: 2017-10-12]
          EEA3 1BFF 4443 04AB B246  A0B6 C634 D042 0F82 5B91
    uid           [ultimate] Cel A. Skeggs <cela [at] mit [dot] edu>

Write access on GitHub is restricted to the hyades-provisioning team.

These security measures exist due to scripts from this repository being used on
trusted systems with /root kerberos tickets or other important auth keys.

# Contact

Current developer: cela. Contact over zephyr (-c hyades) or email @mit.edu.
