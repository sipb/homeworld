# setup.yaml documentation

This file documents the format of the `setup.yaml` file used by Homeworld to configure a Homeworld cluster.

## High-Level Overview

The `setup.yaml` file is a document formatted in YAML, containing a set of sections that have different meanings. You
can refer to [the sample setup.yaml file](../platform/spire/resources/setup.yaml) to get a sense of what it looks like.

These sections are:

 * `cluster`: configuration of the important names in the cluster, such as the domain names and mirrors to use.
 * `vlan`: configuration of trunked VLANs.
 * `addresses`: configuration of the address ranges to use for the cluster.
 * `dns-upstreams`: configuration of the DNS servers.
 * `dns-bootstrap`: manual configuration of temporary DNS entries.
 * `root-admins`: configuration of administrative access.
 * `nodes`: a list of the physical nodes within the cluster.

## Cluster Names

Sample section:

    cluster:
      external-domain: mit.edu
      internal-domain: hyades.local
      etcd-token: <unique-token>
      kerberos-realm: ATHENA.MIT.EDU
      mirror: debian.csail.mit.edu/debian
      user-grant-domain: homeworld.mit.edu
      user-grant-email-domain: MIT.EDU

Descriptions:

 * `external_domain`: the publicly-addressible domain under which the nodes of the cluster can be resolved.

   If this is, for example, `mit.edu`, a node named `rhombi` would be addressible under the hostname `rhombi.mit.edu`.

   Note that these DNS entries do not necessarily need to exist, but it'll be easier if they do.

 * `internal_domain`: the internal domain under which internal DNS addresses should be placed.

   For example, if this is `hyades.local`, then any domain `<service>.<namespace>.svc.hyades.local` would resolve to the
   service IP of the service `<service>` within the namespace `<namespace>` within the cluster. These domains should not
   be publicly addressible, because the IP addresses that they refer to will not be publicly addressible either.

 * `etcd-token`: an opaque token describing your particular cluster, to be used to identify individual etcd clusters.

   This should be globally unique; pick something relating to your cluster that nobody else would choose to use.

   For example, you might pick `mit-sipb-prod-2019-11-24` if your organization was called "MIT SIPB" and was deploying
   Homeworld as a production cluster on 2019-11-24. Note that this format is an example, and you can pick anything that
   you can reasonably believe is unique. (And for testing environments, even that doesn't matter.)

 * `kerberos-realm`: if you are using Kerberos authentication to control access to the cluster, this specifies the realm
   that should be used.

   For example, if you use `ATHENA.MIT.EDU`, then this will work with Kerberos principals of the format
   `<username>@ATHENA.MIT.EDU`.

 * `mirror`: the Debian mirror to use to install the nodes.

   You can pick something like `deb.debian.org/debian` if you want to be generic, or find a local mirror (like
   `debian.csail.mit.edu/debian`) if you want your installs to go faster.

 * `user-grant-domain`: the domain at which the user grant website should be hosted.

   The user grant website is the website that provides authorized users (such as users with an MIT client certificate)
   with a kubectl configuration that allows them to connect to the Homeworld cluster. This should be an accessible
   external hostname (which does not need to be under `external-domain`) that can be pointed at the cluster. In
   practice, since we don't have ingress yet, this should just be set to something you eventually want to use as the
   public hostname.

 * `user-grant-email-domain`: the domain under which client certificates should be checked for authentication by the
   user grant website.

   For example, if your client certificates come with an Subject that includes an email field of the form
   `<username>@MIT.EDU`, you would use `MIT.EDU` in this field. Scanning other fields of client certificates, or using
   other forms of authentication besides client certificates are not yet supported.

## VLAN trunk configuration

Sample section:

    vlan: 612

This is an optional section; in most cases it should not be included. It only needs to be included if your network
configuration uses 802.1Q VLANs _and_ passes them through to individual nodes via VLAN trunk ports. In that case, you
should set the field to the correct VLAN number that your nodes should attach to for the purposes of accessing the
public internet.

## Address configuration

Sample section:

    addresses:
      cidr-nodes: 18.4.60.0/23
      cidr-pods: 172.18.0.0/16
      cidr-services: 172.28.0.0/16
      service-api: 172.28.0.1
      service-dns: 172.28.0.2

If you are not already familiar with CIDR notation, please
[familiarize yourself](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation).

Descriptions:

 * `cidr-nodes`: the publicly-addressible (probably) subnet that the nodes will be directly attached to.

   The gateway and subnet mask for the nodes will be configured based on this subnet. Not all addresses on this subnet
   must be assigned to your nodes; they can co-exist with other servers.

 * `cidr-pods`: the privately-addressible subnet that pod IP addresses should be allocated out of.

   This should normally be a subnet in one of the private-use address ranges. It should not overlap with any other use,
   because nodes will be allocated their own subnets out of this range, and assign their own addresses out of those
   subnets, so you should assume that any and all of these addresses will be allocated automatically.

 * `cidr-services`: the privately-addressible subnet that service IP addresses should be allocated out of.

   This should be a subnet like `cidr-pods`, but can be smaller, because you will generally have many fewer services
   than pods. These are allocated to services across the cluster as needed.

 * `service-api`: the service IP address for the Kubernetes apiserver within the cluster.

   This must always be within the service subnet, and is usually the first address, but can be anything within the range
   that you want.

 * `service-dns`: the service IP address for the DNS addon service within the cluster.

   This must always be within the service subnet, and is usually the second address, but can be anything within the range
   that you want.

## DNS upstream configuration

Sample section:

    dns-upstreams:
      - 18.70.0.160
      - 18.71.0.151
      - 18.72.0.3

This is a list of the local DNS servers that the cluster's nodes should use for upstream DNS. They will be used to
populate /etc/resolv.conf on the servers.

## DNS bootstrap configuration

Sample section:

    dns-bootstrap: {}

This section does not usually need to be filled out. It can optionally be used to force resolution of DNS names to
particular IP addresses, such as the following:

    dns-bootstrap:
      hostname.sample.mit.edu: 18.0.0.1

This example would add an entry to `/etc/hosts` on every node to override the IP address of `hostname.sample.mit.edu` to
be the address `18.0.0.1`. This used to be more useful, but now entries are automatically added to this for the nodes
within the cluster, so it is more useless.

## Root administrator configuration

Sample section:

    root-admins:
      - example/root@ATHENA.MIT.EDU

The root administrators in a cluster are the administrators who are granted complete access -- that is, they can
directly SSH into any node and do whatever they want. This is a list of the Kerberos principals of the root admins.

If this section is empty, it means that Kerberos administrative authentication will be disabled.

Note that, in all cases, root admins can also establish access by simply having the disaster recovery key. This is the
primary way to establish access when Kerberos authentication is not involved.

## Node configuration

Sample section:

    nodes:
      - hostname: master-hostname
        ip: 18.0.0.2
        kind: master

      - hostname: worker-hostname
        ip: 18.0.0.3
        kind: worker

      - hostname: supervisor-hostname
        ip: 18.0.0.4
        kind: supervisor

This is a list of the nodes that are part of the cluster. When nodes are configured and installed, this mapping will be
used to determine what mode they should be placed in, and the supervisor and master node IP addresses will be
substituted into various configuration files.

In theory, you should be able to update this section to change, add, or remove nodes, but this is not fully supported at
this time.

Each entry has the following fields:

 * `hostname`: the hostname (under the external domain specified previously) that the node should be accessible at. This
   does not actually need to be configured in DNS, because it will be coded into /etc/hosts on each node for the time
   being.

 * `ip`: the statically-assigned IP address for this node. This address must be the same address that will be entered
   when running the installer for the node.

 * `kind`: the type of node that this should be.

The kinds of nodes are:

 * `supervisor`: a node that is not part of the Kubernetes cluster proper, but assists with its setup and
   reconfiguration.

   Only a single supervisor node is supported at this time.

   A supervisor node does not need to be up for the regular operations of the cluster, but will need to be up at least
   intermittently to allow key renewal to occur on the other nodes.

 * `master`: a node that runs the management services of the Kubernetes cluster. These nodes do not run user-provided
   code, but do run a full Kubernetes node stack, in addition to the management services.

 * `worker`: a node that runs user containers for the Kubernetes cluster.

The number of master nodes should generally be an odd number, because they will need quorum for their etcd services to
work. Usually three nodes is good for a production cluster, because that will allow any one master node to fail without
losing quorum.

## Tracking setup.yaml

As discussed in the cluster deployment documentation, you should be storing your setup.yaml in a shared Git repository,
so that everyone running your cluster understands the current state correctly.
