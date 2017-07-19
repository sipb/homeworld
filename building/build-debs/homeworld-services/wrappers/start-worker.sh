#!/bin/bash
set -e -u

echo "starting worker services..."

systemctl daemon-reload

systemctl start rkt-api
systemctl enable rkt-api
systemctl start kubelet
systemctl enable kubelet
systemctl start kube-proxy
systemctl enable kube-proxy

echo "services started and enabled!"
