#!/bin/sh
echo "launching postinstall"

BUILDDATE="$1"
GIT_HASH="$2"

. /usr/share/debconf/confmodule

set -e -u

if [ "$BUILDDATE" = "" ]
then
    echo "invalid build date" 1>&2
    exit 1
fi

if [ "$GIT_HASH" = "" ]
then
    echo "invalid git hash" 1>&2
    exit 1
fi

in-target systemctl mask nginx.service

# install packages
cp /homeworld-*.deb /target/
in-target dpkg --install /homeworld-*.deb
rm /target/homeworld-*.deb
in-target apt-get update
in-target apt-get install --fix-broken --yes homeworld-keysystem homeworld-prometheus-node-exporter \
    homeworld-services homeworld-bootstrap-registry homeworld-prometheus homeworld-auth-monitor

in-target systemctl enable keyclient.service update-keyclient-config.timer prometheus-node-exporter.service

mkdir -p /target/etc/homeworld/keyclient/
mkdir -p /target/etc/homeworld/config/
cp /keyservertls.pem /target/etc/homeworld/keyclient/keyservertls.pem
cp /keyclient-*.yaml /target/etc/homeworld/config/
cp /keyclient-base.yaml /target/etc/homeworld/config/keyclient.yaml
cp /sshd_config.new /target/etc/ssh/sshd_config

cat >/tmp/token.template <<EOF
Template: homeworld/asktoken
Type: string
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
        mkdir -p /target/root/.ssh/
        cp /authorized.pub /target/root/.ssh/authorized_keys
    else
        echo "$RET" > /target/etc/homeworld/keyclient/bootstrap.token
    fi
fi

echo "ISO used to install this node generated at: ${BUILDDATE}" >>/target/etc/issue
echo "Git commit used to build the version: ${GIT_HASH}" >>/target/etc/issue
echo "SSH host key fingerprints: (as of install)" >>/target/etc/issue
for x in /target/etc/ssh/ssh_host_*.pub
do
    in-target bash -c "ssh-keygen -l -f ${x#/target} >>/etc/issue"
    in-target bash -c "ssh-keygen -l -E md5 -f ${x#/target} >>/etc/issue"
done
echo >>/target/etc/issue
