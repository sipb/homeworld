package csr

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"util/wraputil"
	"strings"
)

// accepts both public and private keys
func BuildSSHCSR(key []byte) ([]byte, error) {
	privkey, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return ssh.MarshalAuthorizedKey(privkey.PublicKey()), nil
	}
	pubkey, err2 := wraputil.ParseSSHTextPubkey(key)
	if err2 == nil {
		if strings.Contains(pubkey.Type(), "cert") {
			return nil, fmt.Errorf("Expected SSH pubkey file to not have certificate type %s", pubkey.Type())
		}
		return ssh.MarshalAuthorizedKey(pubkey), nil
	}
	return nil, fmt.Errorf("could not parse key as pubkey or privkey: %s (as privkey) / %s (as pubkey)", err, err2)
}

// only accepts private keys, because there are no public key files in TLS
func BuildTLSCSR(privkey []byte) ([]byte, error) {
	key, err := wraputil.LoadRSAKeyFromPEM(privkey)
	if err != nil {
		return nil, err
	}

	template := &x509.CertificateRequest{
		Subject:            pkix.Name{CommonName: "invalid-cn-temporary-request"}, // should be replaced by actual subject on server
		SignatureAlgorithm: x509.SHA256WithRSA,                                    // TODO: ensure that server ignores this properly
	}

	der, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}), nil
}
