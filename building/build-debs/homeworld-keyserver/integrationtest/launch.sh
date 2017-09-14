#!/bin/bash
set -e -u
set -o pipefail

cd "$(dirname "$0")"
killall -9 keyclient || true
killall -9 keyserver || true
rm -rf client server admin
mkdir admin
cp -R client-template client
cp -R server-template server

# server cert
openssl genrsa -out server/authorities/server.key 2048
cat >server/authorities/server.cnf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:TRUE
extendedKeyUsage = clientAuth,serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1=localhost
IP.1=127.0.0.1
EOF
openssl req -x509 -new -key server/authorities/server.key -out server/authorities/server.pem -subj "/CN=localhost-cert" -config server/authorities/server.cnf -extensions v3_req
cp -T server/authorities/server.pem client/server.pem

# ssh_host_ca authority
ssh-keygen -f server/authorities/ssh_host_ca -P ""

# etcd-client authority
openssl genrsa -out server/authorities/etcd-client.key 2048
openssl req -x509 -new -key server/authorities/etcd-client.key -out server/authorities/etcd-client.pem -subj "/CN=etcd-ca"

# granting authority
openssl genrsa -out server/authorities/granting.key 2048
openssl req -x509 -new -key server/authorities/granting.key -out server/authorities/granting.pem -subj "/CN=grant-ca"

# admin cert
openssl genrsa -out admin/auth.key 2048
openssl req -new -key admin/auth.key -out admin/auth.csr -subj "/CN=admin-test"
openssl x509 -req -in admin/auth.csr -CA server/authorities/granting.pem -CAkey server/authorities/granting.key -CAcreateserial -out admin/auth.pem

# prepopulated ssh pubkey
ssh-keygen -f client/ssh_host_rsa_key -P ""

(cd server && ../../keyserver server.yaml 2>&1 | tee server.log) &

curl -sS --cacert client/server.pem --key admin/auth.key --cert admin/auth.pem --data-binary '[{"api": "bootstrap", "body": "localhost-test"}]' https://127.0.0.1:20557/apirequest | cut -d '"' -f 2 >client/bootstrap.token

echo "Token acquired"

(cd client && ../../keyclient client.yaml 2>&1 | tee client.log) &

echo "Letting system resolve to a stable state... (40 seconds)"
sleep 40
echo "Stable state hopefully reached: terminating server and client"

killall keyclient
killall keyserver
wait

echo "TODO: verify stable state"
