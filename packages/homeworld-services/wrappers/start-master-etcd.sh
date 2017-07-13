#!/bin/bash
set -e -u

echo "starting master etcd service..."

systemctl daemon-reload

systemctl start etcd
systemctl enable etcd

echo "services started and enabled!"
