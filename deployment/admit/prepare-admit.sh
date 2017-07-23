#!/bin/bash
set -e -u

if [ "${1:-}" = "" ] || [ "${2:-}" = "" ] || [ "${3:-}" = "" ]
then
	echo "Usage: $0 <secretdir> <admitserver> <hostname>" 1>&2
	exit 1
fi

SECRETDIR="$1"
ADMITSERVER="$2"
HOSTNAME="$3"

ADMIT_KEY="./admit-client.key"
ADMIT_CERT="./admit-client.cert"
ADMIT_CSR="./admit-client.csr"

if [ ! -e "$ADMIT_KEY" ] || [ ! -e "$ADMIT_CERT" ]
then
	openssl genrsa -out "$ADMIT_KEY" 2048
	openssl req -new -key "$ADMIT_KEY" -subj '/CN=admit-autogen-$USER' -out "$ADMIT_CSR"
	openssl x509 -req -in "$ADMIT_CSR" -CA "$SECRETDIR/bootstrap_client_ca.pem" -CAkey "$SECRETDIR/bootstrap_client_ca.key" -CAcreateserial -out "$ADMIT_CERT" -days 1
	rm -f "$ADMIT_CSR"
fi

curl --cacert "$SECRETDIR/admission.pem" --cert "$ADMIT_CERT" --key "$ADMIT_KEY" "https://$ADMITSERVER:2557/bootstrap/$HOSTNAME"
