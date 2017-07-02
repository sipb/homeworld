#!/bin/bash

set -e -u

if [ "${1-}" = "" ] || [ "${2-}" = "" ] || [ "${3-}" = "" ] || [ "${4-}" = "" ]
then
	echo "Usage: $0 <host> <keytab> <acl> <ca> [package]" 1>&2
	echo "  NOTE: The keytab is not automatically rotated!" 1>&2
	exit 1
fi

HOST=$1
KEYTAB=$2
ACL=$3
CA=$4
PACKAGE=${5:-$(dirname "$0")/hyades-authserver_0.1.4-1_amd64.deb}

if [ ! -e "$KEYTAB" ]
then
	echo "Could not find keytab." 1>&2
	exit 1
fi

if [ ! -e "$ACL" ]
then
	echo "Could not find acl." 1>&2
	exit 1
fi

if [ ! -e "$CA" ] || [ ! -e "$CA.pub" ]
then
	echo "Could not find ca and ca.pub." 1>&2
	exit 1
fi

if [ ! -e "$PACKAGE" ]
then
	echo "Could not find package." 1>&2
	exit 1
fi

if ssh "root@${HOST}" test ! -e /etc/krb5.keytab
then
	echo "Uploading keytab..."
	scp "$KEYTAB" "root@${HOST}:/etc/krb5.keytab"
else
	echo "Keytab already exists -- skipping."
fi

scp "${PACKAGE}" "root@${HOST}:/root/hyauth-pkg-install.deb"
scp "${ACL}" "root@${HOST}:/root/.k5login"
# python3 is needed by the admission server
ssh "root@${HOST}" 'apt-get install krb5-user python3 openssl && dpkg -i /root/hyauth-pkg-install.deb && rm /root/hyauth-pkg-install.deb'
scp "${ACL}" "root@${HOST}:/home/kauth/.k5login"
scp "${CA}" "root@${HOST}:/home/kauth/ca_key"
scp "${CA}.pub" "root@${HOST}:/home/kauth/ca_key.pub"
ssh "root@${HOST}" 'chown kauth /home/kauth/ca_key'
ssh "root@${HOST}" 'systemctl restart ssh'

echo "Finished!"
