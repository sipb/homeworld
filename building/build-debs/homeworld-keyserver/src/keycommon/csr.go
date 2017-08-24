package keycommon

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"crypto/rand"
	"crypto/x509/pkix"
	"golang.org/x/crypto/ssh"
	"fmt"
	"wraputil"
)

// accepts both public and private keys
func BuildSSHCSR(key []byte) ([]byte, error) {
	privkey, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return ssh.MarshalAuthorizedKey(privkey.PublicKey()), nil
	}
	pubkey, _, _, rest, err2 := ssh.ParseAuthorizedKey(key)
	if err2 == nil {
		if len(rest) > 0 {
			return nil, fmt.Errorf("Extraneous text after SSH pubkey")
		}
		return ssh.MarshalAuthorizedKey(pubkey), nil
	}
	return nil, fmt.Errorf("could not parse key as pubkey or privkey: %s (as privkey) / %s (as pubkey)", err, err2)
}

func BuildTLSCSR(privkey []byte) ([]byte, error) {
	keydata, err := wraputil.LoadSinglePEMBlock(privkey, []string{"RSA PRIVATE KEY"}) // TODO: support other formats
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keydata)
	if err != nil {
		return nil, err
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "invalid-cn-temporary-request"}, // should be replaced by actual subject on server
		SignatureAlgorithm: x509.SHA256WithRSA, // TODO: ensure that server ignores this properly
	}

	der, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}), nil
}
