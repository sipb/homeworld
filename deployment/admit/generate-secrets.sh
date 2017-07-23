#!/bin/bash
set -e -u

if [ "${1-}" = "" ] || [ "${2-}" = "" ]
then
	echo "Usage: $0 <secretdir> <admithostname>" 1>&2
	exit 1
fi

SECRETDIR="$1"
ADMITHOSTNAME="$2"

function gen_ssl_keypair() {
	KEY="$SECRETDIR/$1.key"
	CERT="$SECRETDIR/$1.pem"
	if [ ! -e "$KEY" ] || [ ! -e "$CERT" ]
	then
		echo "Generating $KEY and $CERT"
		openssl genrsa -out "$KEY" 2048
		openssl req -new -x509 -key "$KEY" -out "$CERT" -days 15 -subj "/CN=$2"
	else
		echo "Already completed generation of $KEY and $CERT"
	fi
}

function gen_ssh_keypair() {
	PRIVKEY="$SECRETDIR/$1"
	PUBKEY="$SECRETDIR/$1.pub"
	if [ ! -e "$PRIVKEY" ] || [ ! -e "$PUBKEY" ]
	then
		echo "Generating $PRIVKEY and $PUBKEY"
		ssh-keygen -f "$PRIVKEY" -N ""
	else
		echo "Already completed generation of $PRIVKEY and $PUBKEY"
	fi
}

gen_ssl_keypair admission "${ADMITHOSTNAME}"
gen_ssl_keypair bootstrap_client_ca "bootstrap-client-ca"

gen_ssh_keypair ssh_user_ca
gen_ssh_keypair ssh_host_ca

echo "Done!"
