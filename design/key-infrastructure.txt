# Scope

This document covers the lifecycle of key management in a Homeworld cluster, with regards to running
cluster nodes. This document does not cover keys used in keysigning that are not specific to a particular
instance of a Homeworld cluster. Authorization, besides for granting of secrets, is not covered here.

## Date and revision

This is the first revision of this document, and is the plan as of 2017-07-25 at 22:10 PM. This document
should be considered tentative and not a finalized design.

# Definitions

*Cluster node*: a server running active code in the cluster that is directly or indirectly involved in
                running user code or handling user data.

*Master node*: a cluster node involved in running Kubernetes masters and the etcd cluster.

*Worker node*: a cluster node involved in directly running user code and storing user data.

*Supervisor node*: a server running non-critical services to help maintain the health of the cluster,
                   but that otherwise aren't involved in serving users directly.

*Homeworld node*: any of the above nodes.

*CA*: a certificate authority, either for SSH or TLS.

*JWT*: JSON Web Tokens, as used internally by kubernetes for its service tokens.

*Cluster admin*: a user capable of some level of authorization to control the infrastructure running Hyades.

*Root admin*: a cluster admin authorized to carry out arbitrary operations on the cluster.

*Cluster user*: a developer who can use the Homeworld cluster for running their code.

# Overview

The key management infrastructure manages the generation, signing, rotation, and distribution of various
public keys, private keys, certificates, and certificate authorities.

Table of certificate authorities and root credentials:

Cred name          | Type   | Holders of certificates   | Issued by    | Privileges granted by issued certs
-------------------|--------|---------------------------|--------------|-------------------------------------------------------------------------------------------------------------
keygranting cert   | TLS CA | Homeworld nodes           | keyserver    | Ability to authenticate as a Homeworld node, but no inherent authorization to receive new and rotated keys.
SSH user           | SSH CA | Root admins               | keyserver    | Ability to SSH into any cluster node as the local root user.
SSH host           | SSH CA | Homeworld nodes           | keyserver    | Ability to authenticate as the represented server hostname to SSH clients.
etcd server        | TLS CA | Master nodes              | keyserver    | Ability to authenticate as part of the etcd cluster, including participation in etcd state management.
etcd client        | TLS CA | Master nodes, root admins | keyserver    | Ability to read and write data in the etcd datastore, and by extension control almost the entire cluster.
kubernetes         | TLS CA | Homeworld nodes           | keyserver    | Ability to authenticate to or as Kubernetes internal services, but not direct authorization to do anything.
serviceaccount key | JWT CA | Containerized services    | kube-ctrlmgr | Ability to authenticate to the Kubernetes apiserver,  but not direct authorization to do anything.

All certificate authorities are held by the keyserver process running on the supervisor nodes,
which grants certificates according to certain policies. Rotation also occurs by talking to the
same keyserver process.

All certificates granted to nodes are long-lived, on the order of thirty days, and all certificates
granted to root admins are short-lived, on the order of four hours.

## Processes

The following processes run on systems in the cluster:

 * The keyclient process runs on each Homeworld node, in two modes:
   - During node initialization, it takes a bootstrapping token and uses it to request initial credentials.
   - Gated by a systemd timer, it wakes up once per day and rotates any keys that need to be rotated.
 * The keyserver daemon runs continously on a supervisor node. It handles a set of processes:
   - Granting long-lived certificates to new Homeworld nodes.
   - Rotating long-lived certificates on existing Homeworld nodes.
   - Granting short-lived certificates to cluster admins. Only kubernetes certificates are available for
     cluster admins who are not root admins, which only grant authentication, not authorization.
 * The kerberos gateway runs continously on a supervisor node. It handles authenticating users via kerberos,
   and then forwarding requests with its own authentication keys to the keyserver daemon, and passing back
   the results or errors. The kerberos gateway is not responsible for handling ACLs or policy, but just for
   tunneling requests and providing verification as to the requestor.

## Bootstrapping Homeworld nodes

When a new server is installed, it will be given a token during the install process (through small
modifications to that process) that was issued by the keyserver (when asked by a root admin) and that
allows a server to obtain particular keys for itself, with permissions as granted by the root admin.

## Policy

Authorization to be granted certificates will be provided by a configuration file on the supervisor node,
which defines the policy by which requests are allowed or denied. This will include both information about
the particular kinds of secrets being managed as well as the particular kerberos principals and keygranting
principals authorized to perform different operations.

The following information will be defined in the policy configuration file:

 * A list of certificate authorities managed by the keyserver
 * A list of principal realms that can grant permissions on the keyserver, including kerberos realms,
   keygranting certificate authorities, and transient bootstrap token realms.
 * A list of authorization groups, each composed of a set of principals, each within a realm.
 * A list of privilege grants, each composed of an authorization group and a corresponding privilege, such
   as authorization to be granted a certain certificate, or authorization to allocate bootstrap tokens.

Each request to the keyserver contains authentication as a specific principal, and a set of operations to
perform. The set of operations will only be accepted and performed if they are all allowed according to
privilege grants to groups that the principal is in.

## Protecting certificate authorities

CAs are protected only by permissions and isolation on the hosts. No user code gets scheduled on any Homeworld
node that has a certificate authority, and worker nodes (which have user code scheduled on them) do not contain
any certificates that would let them directly access the infrastructure layer of the cluster.

## Implementation

This code is handled by a set of packages:

 * homeworld-keyclient: contains binaries and systemd timers for the keyclient system, miscellaneous scripts.
 * homeworld-keyserver: contains binaries and systemd services for the keyserver and kerberos gateway.
 * homeworld-admin-tools: contains scripts for configuring and uploading the keyserver, requesting bootstrap
   tokens, requesting short-lived certificates, and modifying keygranting policy.
 * homeworld-knc: contains generic binaries for kerberized netcat.

Modifications to the installation process will be added so that the bootstrap token can be entered during
debian installation, including scripts to invoke the keyclient automatically when the server first starts,
which will be included in the homeworld-keyclient package.

Both keyclient and keyserver binaries will be written in Go, along with the non-knc component of the kerberos
gateway. The admin tools will be written in bash, and use curl for speaking HTTPS to the keyserver, and use
knc for speaking to the kerberos gateway. Shell scripts to help generate certificate authorities will be
written in bash.

## Future directions

 * We plan to migrate to a system not based on kerberos authentication for cluster admins.
   - This might be OAuth along with a set of escape hatches.
 * We plan to support kerberos authentication for cluster users long-term.
 * We hope to secure certificate authorities within hardware security managers, to limit
   damage of server compromise.
 * We plan to have a server tracking database that, among other things, helps keep track
   of which servers are allowed to still have their keys renewed.
