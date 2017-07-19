#!/bin/bash

set -e -u

if [ "${1-}" = "" ] || [ "${2-}" = "" ] || [ "${3-}" = "" ] || [ "${4-}" = "" ]
then
	echo "Usage: $0 <host> <keytab> <acl> <ca>" 1>&2
	echo "  NOTE: The keytab is not automatically rotated!" 1>&2
	exit 1
fi

HOST=$1
KEYTAB=$2
ACL=$3
CA=$4

for file in "$KEYTAB" "$ACL" "$CA" "$CA.pub"
do
	if [ ! -e "$file" ]; then echo "Could not find $file." 1>&2; exit 1; fi
done

if ssh "root@${HOST}" test ! -e /etc/krb5.keytab
then
	echo "Uploading keytab..."
	scp "$KEYTAB" "root@${HOST}:/etc/krb5.keytab"
else
	echo "Keytab already exists -- skipping."
fi

scp "${ACL}" "root@${HOST}:/root/.k5login"
# python3 and openssl are needed by the admission server
ssh "root@${HOST}" 'apt-get update && apt-get install python3 openssl homeworld-authserver'
scp "${ACL}" "root@${HOST}:/home/kauth/.k5login"
scp "${CA}" "root@${HOST}:/home/kauth/ca_key"
scp "${CA}.pub" "root@${HOST}:/home/kauth/ca_key.pub"
ssh "root@${HOST}" 'chown kauth /home/kauth/ca_key && systemctl restart ssh'

echo "Finished deploying authserver."
