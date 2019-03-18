#!/bin/bash
set -e -u

source /etc/homeworld/config/cluster.conf
source /etc/homeworld/config/local.conf

SRVOPT=()

# TODO: CHANGE THIS
SRVOPT+=(--authorization-mode=AlwaysAllow)
# number of api servers
SRVOPT+=(--apiserver-count "${APISERVER_COUNT}")
# public addresses
SRVOPT+=(--bind-address=0.0.0.0 --advertise-address="${HOST_IP}")
# IP range
SRVOPT+=(--service-cluster-ip-range "${SERVICE_CIDR}")
# use standard HTTPS port for secure port
SRVOPT+=(--secure-port=443)
# etcd cluster to use
SRVOPT+=(--etcd-servers="${ETCD_ENDPOINTS}")
# don't allow privileged containers: don't allow this kind of thing
SRVOPT+=(--allow-privileged=true)
# disallow anonymous users
SRVOPT+=(--anonymous-auth=false)
# various plugins for limitations and protection
SRVOPT+=(--admission-control='NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota,DenyEscalatingExec,SecurityContextDeny')
# authenticate clients properly
SRVOPT+=(--client-ca-file=/etc/homeworld/authorities/kubernetes.pem)
# do HTTPS properly
SRVOPT+=(--tls-cert-file=/etc/homeworld/keys/kubernetes-master.pem --tls-private-key-file=/etc/homeworld/keys/kubernetes-master.key)
# make sure account deletion works
SRVOPT+=(--service-account-lookup)
# no cloud provider
SRVOPT+=(--cloud-provider=)
# authenticate the etcd cluster to us
SRVOPT+=(--etcd-cafile /etc/homeworld/authorities/etcd-server.pem)
# authenticate us to the etcd cluster
SRVOPT+=(--etcd-certfile /etc/homeworld/keys/etcd-client.pem --etcd-keyfile /etc/homeworld/keys/etcd-client.key)
# disallow insecure port
SRVOPT+=(--insecure-port=0)
# authenticate kubelet to us
SRVOPT+=(--kubelet-certificate-authority /etc/homeworld/authorities/kubernetes.pem)
# authenticate us to kubelet
SRVOPT+=(--kubelet-client-certificate=/etc/homeworld/keys/kubernetes-master.pem --kubelet-client-key=/etc/homeworld/keys/kubernetes-master.key)
# let controller manager's service tokens work for us
SRVOPT+=(--service-account-key-file=/etc/homeworld/keys/serviceaccount.key)

exec /usr/bin/hyperkube kube-apiserver "${SRVOPT[@]}"
