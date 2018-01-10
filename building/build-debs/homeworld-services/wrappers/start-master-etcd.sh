#!/bin/bash
set -e -u

echo "starting master etcd service..."

systemctl daemon-reload

systemctl start etcd
systemctl enable etcd
systemctl start etcd-metrics-exporter
systemctl enable etcd-metrics-exporter

echo "services started and enabled!"
