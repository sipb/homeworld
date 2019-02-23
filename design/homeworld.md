This file is an overview of the architecture of Homeworld. You should read it.

# Background

Things you should understand at a basic level, by at least skimming the
following Wikipedia articles:

 * [Kernels](https://en.wikipedia.org/wiki/Kernel_(operating_system))
 * Virtual Machines (read about [hardware virtualization](https://en.wikipedia.org/wiki/Hardware_virtualization))
 * Containers (read about [operating-system-level virtualization](https://en.wikipedia.org/wiki/Operating-system-level_virtualization))
 * [IP addresses](https://en.wikipedia.org/wiki/IP_address)
 * [IP Subnets](https://en.wikipedia.org/wiki/Subnetwork)
 * [DNS](https://en.wikipedia.org/wiki/Domain_Name_System)
 * [Quorum (distributed systems)](https://en.wikipedia.org/wiki/Quorum_(distributed_computing))
 * [Certificate Authority](https://en.wikipedia.org/wiki/Certificate_authority)

You should be able to define each of these and why they are important, though
not anything about how they function in Hyades.

# Motivation

SIPB runs two existing services that the Hyades project intends to replace:
Scripts and XVM. Scripts is a web hosting service that allows users to host
static websites and CGI scripts on shared infrastructure, by serving pages out
of an Athena AFS locker. XVM is a virtual machine hosting service that allows
users to launch virtual machines.

Both of these services are showing their age, and SIPB wants to replace them.

Hyades is designed to be a reliable, secure, and maintainable platform that
can run websites, webapps, MATLAB jobs, and other services and tasks. It will
hand out resources from a shared pool to arbitrary members of the MIT
community, according to assigned quota.

Although it is a SIPB volunteer service, and provides no SLA, it is still
designed to be as reliable as possible -- and avoid situations where an admin
needs to respond to a problem at 3am.

The software used to run the Hyades cluster is the Homeworld cluster management
project.

# Types of Servers

A Homeworld cluster has three categories of servers:

  * A small, odd number of master nodes, which coordinate the cluster.
  * A large number of worker nodes, which run tasks for the cluster.
  * One supervisor node, which is nonessential and runs administrative tasks
    for the cluster.

# Subsystems

This section will give an overview of the different parts of the Homeworld
codebase, organized by directory.

  * Build system: we need a variety of subsystems to help us build and deploy
    a Homeworld cluster properly.

    - chroot system: in order to isolate built Homeworld binaries from the
      messiness of developer machines, it includes a system to generate a
      Homeworld-specific chroot and allow commands to be run within it.

      Location: the files in the root of the repository with "chroot" in their
      names.

    - upstream source cache: upstream projects provide us with source code and
      compiled binaries. we can't trust the compiled binaries for security
      reasons, so we need to build our own binaries whenever possible. we'd
      like to avoid depending on their code distribution websites being online,
      so we cache our own copies of their code ourselves. This also allows us
      to be sure that the upstream source code doesn't change without us
      realizing.

      We have a set of scripts that can automatically pull down the upstream
      source files from their original sources, and compare them to our cached
      versions. We store the upstream archives in a separate git repository,
      at https://github.com/sipb/homeworld-upstream so that our main git
      repository doesn't become massive due to the size of the included
      binaries.

      Location: everything with "upstream" in the name under building/.

    - snapshot cache: in many cases, containers for Homeworld are built from
      debian packages served by snapshot.debian.org. To speed up the build
      process, a local nginx-based cache is provided, which serves binaries
      from the upstream source cache.

      Location: parts of the chroot code, upstream code, and the
      snapshot-cache-nginx file.

    - glass build system: one of the key problems that Homeworld solves is the
      problem of rebuilding upstream source code in a repeatable manner. This
      doesn't fit well with many existing build systems, which are either
      designed for building a single project (e.g. Make, CMake, Bazel) or an
      entire operating system (e.g. the Debian build system). As such, we
      provide our own, fairly simple build system, which is called 'glass'.

      Location: building/glass

    - apt branch system: developers need to be able to upload code to central
      locations, where the code can then be pulled back down by various parts
      of a cluster, without interfering with each others' work.

      An "apt branch" is a partial URL, like "homeworld.celskeggs.com/test01",
      which specifies the base path for downloading debian packages, ACIs, et
      cetera. These are uploaded by glass, and downloaded by a variety of
      different components, such as through apt-get and the bootstrap registry.

      Location: building/apt-branch-config, part of glass, various places that
      code is downloaded.

    - continuous integration system: to ensure that pull requests don't break
      the build, we automatically run a complete build of all components on
      each submitted pull request. This is done via CircleCI. This doesn't
      include running any actual tests, but just building the code.

      Location: .circleci/config.yml, enter-chroot-ci.sh

  * Platform: the basic general-purpose components on which the rest of
    Homeworld is based.

    - debian: we use debian as the basis for all of our physical servers.

      Configuration is in homeworld-admin-tools -- see the preseed file and the
      postinstall script.

    - systemd: we use debian's default init system, systemd, to manage services
      running on the physical servers.

    - go: we build our own golang binaries, because debian doesn't come with a
      new enough version.

      Component: helper-go

    - acbuild: we build our own acbuild binaries

      Component: helper-acbuild

    - debian containers: we build containers that include debian installations
      for use as a basis for other containers. the installations used here are
      downloaded from snapshot.debian.org, or more likely the local mirror of
      it.

      Components: debian, debian-mini, debian-micro

    - apt configuration: we provide configuration as a debian package for our
      apt repositories. Installing this package provides the ability to
      download additional debian packages.

      Component: homeworld-apt-setup

    - spire/administrative tools: we have a custom tool, spire, which is used
      for deploying and managing clusters. this component includes a number of
      critical pieces of system configuration and setup encoded both in the
      included resources and the management code.

      Importantly, a number of other components have configuration embedded in
      this component's resources/clustered directory.

      Components: homeworld-admin-tools, homeworld-debian-iso

    - prometheus: we run a monitoring system called prometheus that collects
      various metrics from around the cluster and synthesizes them into a
      single time-ordered database. this is used by developers for
      understanding the state of the cluster, and by spire for verifying that
      a cluster is functioning properly.

      Component: homeworld-prometheus

    - node metrics: we run the prometheus node exporter agent to provide info
      on what each physical node is doing, with regards to memory, cpu, disk,
      services, et cetera.

      Component: homeworld-prometheus-node-exporter

    - container registry: in order to have ACI containers available within the
      cluster, we run a small nginx server on the supervisor node that gets
      mapped to homeworld.private, and serves the redirection page needed by
      ACI autodiscovery.

      Component: homeworld-bootstrap-registry

    - pull monitor: to make sure that the cluster can successfully download and
      run containers, a service runs on every node, which constantly performs
      these operations. Prometheus aggregates the resulting measurements.

      Components: homeworld-pull-monitor, pullcheck
      Configuration is in homeworld-services.

  * Authentication Layer:

    - keysystem: we have a custom subsystem that manages authentication and key
      distribution for the cluster infrastructure itself. see
      key-infrastructure.md for some outdated documentation on this.

      Components: homeworld-keysystem, homeworld-knc
      Additonal location: building/components/sources-shared/src

    - ssh: to allow administrators to directly connect to the cluster's nodes,
      the keysystem is used to configure SSH certificates for bidirectional
      authentication of the cluster.

      Configuration is in homeworld-admin-tools.

    - auth monitor: in order to track the status of the keysystem and the
      ability for admins to SSH into the cluster's infrastructure, a dedicated
      component runs on the supervisor and continually checks the status of
      these parts of the infrastructure.

      Component: homeworld-auth-monitor

  * Disk Cluster:

    - ceph: not yet merged to master. Ceph will, once configured, manage the
      disks attached to the cluster.

    - faraday: not yet merged to master. Provides an encrypted overlay network
      to help mitigate the problems with Ceph's broken authentication system.

  * Compute Cluster:

    - etcd: in order to have a cluster with no single point of failure, we need
      a central fault-tolerance key/value store, which can be used by
      kubernetes for storing cluster state.

      Component: homeworld-etcd
      Configuration is in homeworld-services.

    - etcd monitor: since all access to the etcd interface is privileged, we
      use a proxy component to expose the non-privileged metrics endpoints of
      etcd.

      Component: homeworld-etcd-metrics-exporter
      Configuration is in homeworld-services.

    - kubernetes: we use kubernetes, a cluster orchestration system, to manage
      the containers running on all nodes. This also helps manage some parts of
      the network cluster. All kubernetes binaries are built into a single
      'hyperkube' single-call binary.

      We interface with rkt, our container manager, through a system called
      rktnetes, which is integrated with the kubelet source on older versions
      of kubernetes. This is deprecated in newer versions, so we plan to switch
      to rktlet instead.

      Component: homeworld-hyperkube
      Configuration is in homeworld-services.

    - kubernetes monitor: we export statistics and state information from the
      kubernetes cluster such that it can be consumed by prometheus, via a
      dedicated service on the supervisor node.

      Component: kube-state-metrics
      Configuration is in homeworld-services -- the kube-related service files,
      the apiserver service, and the homeworld-autostart service.

    - rkt: rather than using docker, which is the best-known container runtime,
      we use a system called 'rkt', which provides similar features, but in a
      more composable and secure way. Specifically, we can use rkt's kvm
      stage1, which gives us better container isolation without the full
      overhead of an entire VM.

      Component: homeworld-rkt

    - setup queue: rather than have spire directly deploy kubernetes objects to
      the cluster, spire deploys object configurations to a queue on the
      supervisor node, which is pulled from by the setup-queue service. This
      allows spire to avoid needing to run any special code after setting up
      the supervisor.

      Configuration is in homeworld-services.

  * Network Cluster:

    - flannel: in order to have a network that all kubernetes pods are
      reachable from, we provide an unencrypted overlay network with a tool
      called flannel. This is launched on each node by kubernetes.

      The homeworld-services component contains configuration that allows rkt
      to use this network.

      Component: flannel
      Configuration is in homeworld-admin-tools.

    - flannel monitor: to confirm that flannel is doing its job, a set of
      services are launched on each node by kubernetes that check communication
      between every pair of nodes. This data is exported to prometheus.

      Component: flannel-monitor
      Configuration is in homeworld-admin-tools.

    - kube-proxy: for cluster-internal load balancing, we use kubernetes's
      built-in on-each-node load balancer.

      Component: homeworld-hyperkube
      Configuration is in homeworld-services.

    - glb: we intend to use github's load balancer for layer 4 load balancing
      from external clients. This is not yet implemented, but see
      https://githubengineering.com/glb-director-open-source-load-balancer/
      for more information.

    - envoy: we intend to use envoy for layer 7 load balancing from external
      clients. This is not yet implemented, but see https://www.envoyproxy.io/
      for more information.

    - dns addon: one of kubernetes's default on-cluster services is a DNS
      service that allows addresses for various cluster systems to be looked
      up as domain names. This runs on the cluster itself.

      Components: kube-dns-main, kube-dns-sidecar, dnsmasq, dnsmasq-nanny
      Configuration is in homeworld-admin-tools.

    - dns monitor: to confirm that the dns addon is working properly, a service
      is launched to monitor it and report back the results to prometheus.

      Component: dns-monitor
      Configuration is in homeworld-admin-tools.
