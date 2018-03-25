#!/bin/bash
set -e -u

# this script is run exactly once, when the cluster is first deployed.

# it has two responsibilities:
#     generate keys
#     update kubernetes secret with generated keys

mkdir keyrings
cd keyrings

echo "generating ceph keys"

ceph-authtool --create-keyring "./mon.keyring"                  --gen-key -n mon.                             --cap mon 'allow *'
ceph-authtool --create-keyring "./client.admin.keyring"         --gen-key -n client.admin         --set-uid=0 --cap mon 'allow *' --cap osd 'allow *' --cap mds 'allow *' --cap mgr 'allow *'
ceph-authtool --create-keyring "./client.bootstrap-osd.keyring" --gen-key -n client.bootstrap-osd             --cap mon 'profile bootstrap-osd'

echo "merging ceph keys"

ceph-authtool "./mon.keyring" --import-keyring "./client.admin.keyring"
ceph-authtool "./mon.keyring" --import-keyring "./client.bootstrap-osd.keyring"

echo "uploading ceph keys"

echo "{\"data\": {\"mon.keyring\": \"$(base64 -w 0 <mon.keyring)\", \"client.admin.keyring\": \"$(base64 -w 0 <client.admin.keyring)\", \"client.bootstrap-osd.keyring\": \"$(base64 -w 0 <client.bootstrap-osd.keyring)\"}}" >secret.patch

curl -v --cacert /var/run/secrets/kubernetes.io/serviceaccount/ca.crt -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" -X PATCH \
     -H "Content-Type: application/strategic-merge-patch+json" "https://kubernetes.default.svc.hyades.local/api/v1/namespaces/${POD_NAMESPACE}/secrets/${SECRET_NAME}" \
     -d "$(cat secret.patch)"

echo "keys uploaded!"
