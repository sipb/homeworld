#!/bin/sh
echo "launching postinstall"

set -e -u

BUILDDATE="$1"
GIT_HASH="$2"
SUPERVISOR_IP="$3"

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

if [ "$SUPERVISOR_IP" = "" ]
then
    echo "invalid supervisor IP" 1>&2
    exit 1
fi

in-target systemctl mask nginx.service

# install packages
cp /homeworld-*.deb /target/
in-target apt-get install --yes apt-transport-https
in-target dpkg --install /homeworld-*.deb
rm /target/homeworld-*.deb
in-target apt-get update
in-target apt-get install --fix-broken --yes homeworld-keysystem homeworld-prometheus-node-exporter \
    homeworld-services homeworld-docker-registry homeworld-prometheus homeworld-auth-monitor

in-target systemctl enable keyclient.service prometheus-node-exporter.service
in-target systemctl enable homeworld-autostart.service

mkdir -p /target/etc/homeworld/keyclient/
mkdir -p /target/etc/homeworld/config/
cp /keyservertls.pem /target/etc/homeworld/keyclient/keyservertls.pem
cp /keyserver.domain /target/etc/homeworld/config/keyserver.domain
cp /sshd_config.new /target/etc/ssh/sshd_config
cat /dns_bootstrap_lines >> /target/etc/hosts

if [ "$SUPERVISOR_IP" = "$(ip -o -f inet addr show scope global up | tr -s " " " " | cut -d ' ' -f 4 | cut -d '/' -f 1)" ]
then
    mkdir -p /target/root/.ssh/
    cp /authorized.pub /target/root/.ssh/authorized_keys
fi

echo "ISO used to install this node generated at: ${BUILDDATE}" >>/target/etc/issue
echo "Git commit used to build the version: ${GIT_HASH}" >>/target/etc/issue
echo "SSH host key fingerprints: (as of install)" >>/target/etc/issue
for x in /target/etc/ssh/ssh_host_*.pub
do
    in-target bash -c "ssh-keygen -l -f ${x#/target} >>/etc/issue"
    in-target bash -c "ssh-keygen -l -E md5 -f ${x#/target} >>/etc/issue"
done
echo "Keygranting key fingerprint: (as of install)" >>/target/etc/issue
in-target bash -c "keyinittoken >>/etc/issue"
echo >>/target/etc/issue
echo >>/target/etc/issue
