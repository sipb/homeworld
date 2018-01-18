#!/bin/bash
set -e -u

echo "starting master services..."

systemctl daemon-reload

# etcd should already be started by start-master-etcd.sh
systemctl start rkt-api
systemctl enable rkt-api
systemctl start rkt-gc.timer
systemctl enable rkt-gc.timer
systemctl start aci-pull-monitor
systemctl enable aci-pull-monitor
systemctl start kubelet
systemctl enable kubelet
systemctl start kube-proxy
systemctl enable kube-proxy
systemctl start apiserver
systemctl enable apiserver
systemctl start kube-ctrlmgr
systemctl enable kube-ctrlmgr
systemctl start kube-scheduler
systemctl enable kube-scheduler

echo "services started and enabled!"
