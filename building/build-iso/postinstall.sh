#!/bin/bash
set -e -u

echo "launching postinstall"

. /usr/share/debconf/confmodule

# install packages
cp /homeworld-*.deb /target/
in-target dpkg -i /homeworld-*.deb
rm /target/homeworld-*.deb

in-target systemctl enable keyclient.service update-keyclient-config.timer

mkdir -p /target/etc/homeworld/keyclient/
mkdir -p /target/etc/homeworld/config/
cp /keyservertls.pem /target/etc/homeworld/keyclient/keyservertls.pem
cp /keyclient-*.yaml /target/etc/homeworld/config/
cp /keyclient-base.yaml /target/etc/homeworld/config/keyclient.yaml

cat >/tmp/token.template <<EOF
Template: homeworld/asktoken
Type: text
Description: Enter the bootstrap token for this server.

Template: homeworld/title
Type: text
Description: Configuring keysystem...
EOF

debconf-loadtemplate homeworld /tmp/token.template

db_settitle homeworld/title

db_input critical homeworld/asktoken
db_go

db_get homeworld/asktoken

if [ "$RET" != "" ]
then
    if [ "$RET" = "manual" ]
    then
        mkdir /target/root/.ssh/
        cp /authorized.pub /target/root/.ssh/authorized_keys
    else
        echo "$RET" > /target/etc/homeworld/keyclient/bootstrap.token
    fi
fi

echo "SSH host key fingerprints: (as of install)" >>/etc/issue
for x in /etc/ssh/ssh_host_*.pub
do
    ssh-keygen -l -f ${x} >>/etc/issue
done
echo >>/etc/issue
