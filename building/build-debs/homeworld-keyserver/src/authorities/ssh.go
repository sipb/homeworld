package authorities

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"math/big"
	"time"
)

type SSHAuthority struct {
	key    ssh.Signer
	pubkey []byte
}

func parseSingleSSHKey(data []byte) (ssh.PublicKey, error) {
	pubkey, _, _, rest, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, err
	}
	if rest != nil && len(rest) > 0 {
		return nil, fmt.Errorf("Trailing data after end of public key file")
	}
	return pubkey, nil
}

func arePublicKeysEqual(pk1 ssh.PublicKey, pk2 ssh.PublicKey) bool {
	if pk1.Type() != pk2.Type() {
		return false
	} else {
		return bytes.Equal(pk1.Marshal(), pk2.Marshal())
	}
}

func LoadSSHAuthority(keydata []byte, pubkeydata []byte) (Authority, error) {
	pubkey, err := parseSingleSSHKey(pubkeydata)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePrivateKey(keydata)
	if err != nil {
		return nil, err
	}
	if !arePublicKeysEqual(pubkey, key.PublicKey()) {
		return nil, fmt.Errorf("Public SSH key does not match private SSH key: %s versus %s", pubkey.Marshal(), key.PublicKey().Marshal())
	}
	return &SSHAuthority{key: key, pubkey: pubkeydata}, nil
}

func (d *SSHAuthority) GetPublicKey() []byte {
	return d.pubkey
}

func certType(ishost bool) uint32 {
	if ishost {
		return ssh.HostCert
	} else {
		return ssh.UserCert
	}
}

func marshalSSHCert(cert *ssh.Certificate) string {
	return fmt.Sprintf("%s %s\n", cert.Type(), base64.StdEncoding.EncodeToString(cert.Marshal()))
}

func (d *SSHAuthority) Sign(request string, ishost bool, lifespan time.Duration, name string, othernames []string) (string, error) {
	pubkey, err := parseSingleSSHKey([]byte(request))
	if err != nil {
		return "", err
	}

	if lifespan < time.Second {
		return "", fmt.Errorf("Lifespan is too short (or nonpositive) for certificate signature.")
	}

	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(64), nil))
	if err != nil {
		return "", err
	}

	cert := &ssh.Certificate{
		Key:             pubkey,
		KeyId:           name,
		Serial:          serialNumber.Uint64(),
		CertType:        certType(ishost),
		ValidAfter:      uint64(time.Now().Unix()),
		ValidBefore:     uint64(time.Now().Add(lifespan).Unix()),
		ValidPrincipals: othernames,
	}

	err = cert.SignCert(rand.Reader, d.key)
	if err != nil {
		return "", err
	}

	return marshalSSHCert(cert), nil
}
