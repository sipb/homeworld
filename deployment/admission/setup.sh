#!/bin/bash
set -e -u

SSH_DIR=/etc/ssh/

cat >server-tmp.cert <<EOCERTIFICATE_HERE
{{SERVER CERTIFICATE}}
EOCERTIFICATE_HERE

for type in ecdsa ed25519 rsa
do
	pub=${SSH_DIR}/ssh_host_${type}_key.pub
	certtemp=${SSH_DIR}/ssh_host_${type}_cert.tmp
	cert=${SSH_DIR}/ssh_host_${type}_cert
	echo "===================== VERIFY WITH ====================="
	sha256sum "${pub}" | cut -d " " -f 1
	echo "===================== VERIFY WITH ====================="
	rm -f -- "${certtemp}"
	wget -nv --tries=1 --ca-certificate=server-tmp.cert "https://{{SERVER IP}}/sign/$(cat "${pub}")" -O "${certtemp}"
	mv -- "${certtemp}" "${cert}"
done

echo "Installing static details..."

rm -f "${SSH_DIR}/ssh_user_ca.pub.tmp"
cat > "${SSH_DIR}/ssh_user_ca.pub.tmp" <<EOSSH_USER_CA_HERE
{{SSH CA}}
EOSSH_USER_CA_HERE
mv "${SSH_DIR}/ssh_user_ca.pub.tmp" "${SSH_DIR}/ssh_user_ca.pub"

rm -f "${SSH_DIR}/sshd_config.tmp"
cat > "${SSH_DIR}/sshd_config.tmp" <<EOSSHD_CONFIG_HERE
{{SSHD CONFIG}}
EOSSHD_CONFIG_HERE
mv "${SSH_DIR}/sshd_config.tmp" "${SSH_DIR}/sshd_config"

wget -nv --tries=1 --ca-certificate=server-tmp.cert "https://{{SERVER IP}}/finish" -O /dev/null

rm -f server-tmp.cert "$0"

systemctl restart ssh

echo "Done!"
