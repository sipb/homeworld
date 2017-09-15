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

echo "About to run setup process..."

../integrationhelper setup

echo "Setup process executed."

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

echo "Verifying stable state..."

../integrationhelper check
