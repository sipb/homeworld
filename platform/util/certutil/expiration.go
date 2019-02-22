package certutil

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"time"

	"github.com/sipb/homeworld/platform/util/wraputil"
)

func CheckSSHCertExpiration(key []byte) (time.Time, error) {
	pubkey, err := wraputil.ParseSSHTextPubkey(key)
	if err != nil {
		return time.Time{}, err
	}
	cert, ok := pubkey.(*ssh.Certificate)
	if !ok {
		return time.Time{}, fmt.Errorf("found public key instead of certificate when checking expiration")
	}
	return time.Unix(int64(cert.ValidBefore), 0), nil
}

func CheckTLSCertExpiration(certdata []byte) (time.Time, error) {
	cert, err := wraputil.LoadX509CertFromPEM(certdata)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}
