# Role-Based Access Control (RBAC) Architecture Overview

This documentation is intended to be a brief overview of the configuration that Homeworld uses for access control, based on Kubernetes's RBAC configuration.

## RBAC overview

At a high level, RBAC in our configuration works as follows:

 * Kubernetes requires both a configured *authentication* scheme and a configured *authorization* scheme.
    * We support certificate authentication and service account token authentication.
    * We support RBAC authorization and Node authorization.
 * The Kubernetes apiservers are in charge of enforcing policies stored as objects in etcd storage.
   * The apiservers automatically create (or recreate) a set of default policies whenever they start up.
   * We provide a set of additional policies that we deploy as part of predeployment (i.e. setup-queue deployment).
   * RBAC policies are *additive*, so additional policies can only grant additional access.
 * Certificates to authenticate to the Kubernetes cluster are granted by the keysystem and by the user-grant service. These certificates encode a username and a set of groups that are then used by the authorization system to decide whether a node has access to particular resource.
 * RBAC authorization is used for almost all authorization, but the Node authorization mode is used to further restrict the ability for Nodes to modify cluster state, by limiting them to the data needed to work with the pods running on them.
 * RBAC works by specifying a set of ClusterRoles and Roles, which specify different kinds of permissible access, and then binding them to users via ClusterRoleBindings and RoleBindings.

## RBAC Users

This is the list of users and username formats that we work with in our cluster.

User name                      | Groups list    | Issued to                                             | Notes
-------------------------------|----------------|-------------------------------------------------------|------------------------------------------------------------------------------------------------------
system:node:[hostname]         | system:node    | The kubelet running on each master or worker node.    | Only the hostname, not the full domain name.
supervisor:[hostname]          | system:masters | The setup-queue on the supervisor.                    | This is also used by the kube-state-metrics and prometheus services on the supervisor.
apiserver:[hostname]           |                | The apiservers for their server certificates.         | This is never used to authenticate to the cluster, only to secure TLS connections to the apiservers.
root:[principal]               | system:masters | Root admins when authenticating via the keysystem.    | The full principal, `user/root@ATHENA.MIT.EDU`, is used.
root:direct                    | system:masters | Root admins when authenticating via keysystem bypass. | The prefix `root:` does not grant access; only the `system:masters` group does.
user:[kerberos]                |                | Users when authenticating via user-grant.             | Only the simple username, `user`, is used.
system:kube-controller-manager |                | The controller managers running on the master nodes.  | This user is used to provision the service accounts actually used by the controller manager.
system:kube-proxy              |                | The proxies running on master and worker nodes.       |
system:kube-scheduler          |                | The schedulers running on the master nodes.           |

### Special groups

These groups are broadly defined, and so not listed above.

 * `system:authenticated`: every user authenticated to the cluster
 * `system:unauthenticated`: anyone connecting anonymously (not enabled in our cluster)

### Service Accounts

Service accounts can be dynamically created and destroyed as part of Kubernetes execution. A selection are listed here, all of which are in the `kube-system` namespace.

Service Account Name      | Issued to                                                    | Notes
--------------------------|--------------------------------------------------------------|--------------------------------------------------------------------------------
[system]-controller       | The control loops in the controller manager.                 | The controller manager is responsible for creating its own service accounts.
horizontal-pod-autoscaler | One of the control loops in the controller manager.          | It's not technically a "controller", so it's named differently.
persistent-volume-binder  | One of the control loops in the controller manager.          | It's not technically a "controller", so it's named differently.
bootstrap-signer          | One of the control loops in the controller manager.          | It's not technically a "controller", so it's named differently.
cloud-provider            | One of the control loops in the controller manager.          | It's not technically a "controller", so it's named differently.
token-cleaner             | One of the control loops in the controller manager.          | It's not technically a "controller", so it's named differently.
flannel                   | The containers actively running the flannel overlay network. |
flannel-monitor           | The containers monitoring the flannel overlay network.       |
kube-dns                  | The containers actively running the kubernetes DNS service.  |
user-grant                | The containers hosting the user-grant service.               |

### Bootstrapping our access policies

Kubernetes installs a set of default policies into the cluster on apiserver startup, but these aren't the complete list of policies that we need.

In order to deploy our own policies, we use the setup queue, which means that it needs to have access even though our policies aren't configured. As such, we grant a special `system:masters`-group user for the supervisor node. This is a group interpreted by the default RBAC policies to provide complete access to absolutely everything, so we only grant it to the root admins and the supervisor.

As things currently stand, the prometheus and kube-state-metrics services also use the same special supervisor access, which is okay for now, but should be changed in the future.

### Providing users with access

User access is automatically granted by having the user-grant service create a rolebinding when it creates the namespace for a user. This works fine, but has the concerning property that it gives the user-grant service a *lot* of power, since it can grant access to *any* namespace. Of course, this service is already concerning, because it can sign a certificate for anyone, as well.

We grant users the `system:admin` permission, scoped only to their namespace of the form `user-[username]`.
