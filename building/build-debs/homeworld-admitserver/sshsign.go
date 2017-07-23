package main

import (
	"golang.org/x/crypto/ssh"
	"bytes"
	"encoding/base64"
	"time"
	"crypto/rand"
	"io/ioutil"
	"fmt"
)

type SSHCertificateAuthority struct {
	ca_key ssh.Signer
	validity_interval time.Duration
}

func LoadSSHCertificateAuthority(filename string, validity_interval time.Duration) (*SSHCertificateAuthority, error) {
	raw_ca_key, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	signing_ca_key, err := ssh.ParsePrivateKey(raw_ca_key)
	if err != nil {
		return nil, err
	}
	return &SSHCertificateAuthority{ signing_ca_key, validity_interval }, nil
}

func (ca *SSHCertificateAuthority) SignHostPubkey(pubkey ssh.PublicKey, keyid string, hostnames []string) (*ssh.Certificate, error) {
	cert := &ssh.Certificate{
		Key:             pubkey,
		KeyId:           keyid,
		CertType:        ssh.HostCert,
		ValidAfter:      uint64(time.Now().Unix()),
		ValidBefore:     uint64(time.Now().Add(ca.validity_interval).Unix()),
		ValidPrincipals: hostnames,
	}

	err := cert.SignCert(rand.Reader, ca.ca_key)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

func (ca *SSHCertificateAuthority) SignHostPubkeys(pubkeys []ssh.PublicKey, keyid string, hostnames []string) ([]*ssh.Certificate, error) {
	var certs []*ssh.Certificate
	for _, pubkey := range pubkeys {
		cert, err := ca.SignHostPubkey(pubkey, keyid, hostnames)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

func parseSSHKeys(data []byte) []ssh.PublicKey {
	var data_out []ssh.PublicKey
	for len(data) > 0 {
		out, _, _, more_data, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			break
		}
		data_out = append(data_out, out)
		data = more_data
	}
	return data_out
}

func (ca *SSHCertificateAuthority) SignEncodedHostPubkeys(pubkeys []byte, keyid string, hostnames []string) ([]byte, error) {
	certs, err := ca.SignHostPubkeys(parseSSHKeys(pubkeys), keyid, hostnames)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("No pubkeys found")
	}
	b := &bytes.Buffer{}
	for _, cert := range certs {
		b.Write(MarshalCert(cert))
	}
	return b.Bytes(), nil
}

func MarshalCert(cert *ssh.Certificate) []byte {
	b := &bytes.Buffer{}
	b.WriteString(cert.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(cert.Marshal())
	e.Close()
	b.WriteByte('\n')
	return b.Bytes()
}
