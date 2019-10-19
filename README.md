# The Homeworld Project

Homeworld is a self-assembling bare-metal distribution of [Kubernetes](https://kubernetes.io/) built by [MIT SIPB](https://sipb.mit.edu/) for our [Hyades cluster](http://hyades.mit.edu/).

# Key features

Disclaimer: Homeworld is unfinished, and not all of these features are fully implemented. Do not trust Homeworld with your critical data or services at this time.

 * Bare-metal: Homeworld handles the entire installation and operation process of a cluster, from providing a customized debian installer to integrating with external authentication mechanisms.
 * Self-assembling: Homeworld is designed to be easily deployable and scalable on bare metal hardware; once the supervisor node is installed, each additional node (master or worker) only requires an IP address and an admission token, and then everything else is automated.
 * Self-contained: Homeworld builds almost everything from source (except Go and Debian), and does not depend on the security of container registries like Docker Hub. Upstream sources are themselves pinned, to help defend against supply-chain attacks.
 * Secure by default: Homeworld strives to operate securely without requiring any additional configuration or tooling.
 * Self-testing: Homeworld's integration tests are (in most cases) the same as the monitoring tools used to ensure that the cluster is functional, so monitoring confirms not just that elements of the system are online, but that they are functional.

# Subfolders

 * platform: the bazel workspace, containing all of the code used in Homeworld except for the chroot scripts.
 * build-chroot: scripts for creating and managing a chroot suitable for building Homeworld, which is required.
 * deploy-chroot: scripts for creating and managing a chroot suitable for deploying and administering Homeworld, which is optional.
 * .circleci: configuration for continuous integration of the Homeworld build process.
 * .jenkins: configuration for continuous integration of the Homeworld build and deployment (i.e. self-assembly) processes.
 * docs: documentation on building, maintaining, and deploying Homeworld.
 * design: somewhat outdated and incomplete documentation on Homeworld's design.

# Contact

Project lead: Cel Skeggs. Contact over SIPB Mattermost or via MIT email.

