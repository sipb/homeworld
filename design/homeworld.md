This file is an overview of the architecture of Homeworld. You should read it, and then read it again.

 -- THIS DOCUMENTATION IS IN PROGRESS --

# Background

Things you should understand at a basic level, by at least skimming the following Wikipedia articles:

 * [Kernels](https://en.wikipedia.org/wiki/Kernel_(operating_system))
 * Virtual Machines (read about [hardware virtualization](https://en.wikipedia.org/wiki/Hardware_virtualization))
 * Containers (read about [operating-system-level virtualization](https://en.wikipedia.org/wiki/Operating-system-level_virtualization))
 * [IP addresses](https://en.wikipedia.org/wiki/IP_address)
 * [IP Subnets](https://en.wikipedia.org/wiki/Subnetwork)
 * [DNS](https://en.wikipedia.org/wiki/Domain_Name_System)
 * [Quorum (distributed systems)](https://en.wikipedia.org/wiki/Quorum_(distributed_computing))
 * [Certificate Authority](https://en.wikipedia.org/wiki/Certificate_authority)

You should be able to define each of these and why they are important, though not anything about how they function in Hyades.

# Motivation

SIPB runs two existing services that the Hyades project intends to replace: Scripts and XVM.

Scripts is a web hosting service that allows users to host static websites and CGI scripts on shared infrastructure, by serving pages out of an Athena AFS locker.

XVM is a virtual machine hosting service that allows users to launch virtual machines.

Both of these services are showing their age, and SIPB wants to replace them.

Hyades is designed to be a reliable, secure, and maintainable platform that can run websites, webapps, matlab jobs, and other services and tasks.
It will hand out resources from a shared pool to arbitrary members of the MIT community, according to assigned quota.

Although it is a SIPB volunteer service, and provides no SLA, it is still designed to be up as much as possible -- and avoid situations where an admin needs to
wake up at 3am to trek over to the datacenter and fix something.

# Major Components

Homeworld is the major subsystem of Hyades, which provides infrastructure, but not a higher level platform.

It has three kinds of servers:

  * A small, odd number of Master Nodes, which coordinate the cluster.
  * A large number of Worker Nodes, which run tasks for the cluster.
  * A small number of Supervisor Nodes, which are nonessential and run administrative tasks for the cluster.

It has a number of major components:

  * Platform: the basic installation, operation, administration, and security management systems that form the basis for Homeworld.
  * Disk Cluster: the cluster used for reliably storing persistent state, both object storage and block storage.
  * Compute Cluster: the cluster used for scheduling and running tasks and services, along with maintaining consistency between different components of the cluster.
  * Network Cluster: the systems used for connecting different portions of the cluster so that they act as a whole.

## Platform

  * Debian: Homeworld uses Debian as the basis for the installed systems and the individual containers.
  * systemd: Homeworld uses systemd to launch and restart individual services on the cluster nodes.
  * KNC: Homeworld uses KNC (kerberized netcat) for authenticating the root administrators to the keysystem using kerberos.
  * Key System: Homeworld uses the keysystem (developed as part of the Homeworld project) to handle administrative and infrastructural authentication within the cluster.
  * SSH: Homeworld uses SSH (and SSH certificates) to let administrators log into the infrastructure directly.
  * Spire: Homeworld uses Spire (developed as part of the Homeworld project) to bootstrap and configure the cluster.

## Disk Cluster

  * Ceph: Homeworld uses Ceph to provide storage of persistent state, both object storage and block storage.
  * Faraday: Homeworld uses Faraday (developed as part of the Homeworld project) to encapsulate Ceph in a secure communication fabric.

## Compute Cluster

  * Consistency: Homeworld uses etcd to achieve quorum on the master nodes, to maintain consistency, even in the face of network partitions.
  * Kubernetes Cluster: Homeworld uses Kubernetes to perform a variety of functions for the Compute Cluster:
    - schedule containers onto individual worker nodes
    - launch new instances of containers when other instances fail
  * rkt: Homeworld uses rkt, a container manager (like docker), to launch and manage containers on the worker nodes.
  * rkt kvm stage1: Homeworld uses rkt's kvm stage1 to provide a layer of isolation around containers, based on lightweight virtual machines.

## Network Cluster

  * Flannel: Homeworld uses Flannel to let each container have a unique IP address, addressible from anywhere in the cluster.
  * Spike: Homeworld uses Spike (also developed as part of the Hyades project) to load-balance service requests from external users outside the cluster.
  * kube-proxy: Homeworld uses the Kube-Proxy component of Kubernetes to load-balance service requests from internal sources within the cluster.
