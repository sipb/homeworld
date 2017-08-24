package keyclient

import (
	"time"
	"golang.org/x/crypto/ssh"
	"fmt"
	"encoding/pem"
	"crypto/x509"
	"io/ioutil"
	"os"
	"keycommon"
	"wraputil"
)

type RequestOrRenewAction struct {
	Mainloop *Mainloop
	InAdvance time.Duration
	API string
	Name string
	CheckExpiration func([]byte) (time.Time, error)
	GenCSR func([]byte) ([]byte, error)
	KeyFile string
	CertFile string
}

func CheckSSHCertExpiration(key []byte) (time.Time, error) {
	pubkey, _, _, rest, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return time.Time{}, err
	}
	if len(rest) > 0 {
		return time.Time{}, fmt.Errorf("Extraneous data after SSH certificate")
	}
	cert, ok := pubkey.(*ssh.Certificate)
	if !ok {
		return time.Time{}, fmt.Errorf("Found public key instead of certificate when checking expiration")
	}
	return time.Unix(int64(cert.ValidBefore), 0), nil
}

func CheckTLSCertExpiration(key []byte) (time.Time, error) {
	blockdata, err := wraputil.LoadSinglePEMBlock(key, []string{"CERTIFICATE"})
	if err != nil {
		return time.Time{}, err
	}
	cert, err := x509.ParseCertificate(blockdata)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

func (ra *RequestOrRenewAction) Perform() error {
	existing, err := ioutil.ReadFile(ra.CertFile)
	if err != nil {
		if os.IsNotExist(err) {
			// not really an error; fall through and always populate
		} else {
			return fmt.Errorf("While trying to check expiration status of certificate: %s", err)
		}
	} else {
		expiration, err := ra.CheckExpiration(existing)
		if err != nil {
			return fmt.Errorf("While trying to check expiration status of certificate: %s", err)
		}
		renew_at := expiration.Add(-ra.InAdvance)
		if renew_at.After(time.Now()) {
			return ErrNothingToDo  // we have a cert and it's not yet time to renew it
		}
		// time to renew!
	}
	if ra.Mainloop.keygrant == nil {
		return errBlockedAction{"No keygranting certificate ready."}
	}
	keydata, err := ioutil.ReadFile(ra.KeyFile)
	if err != nil {
		return err
	}
	csr, err := ra.GenCSR(keydata)
	if err != nil {
		return err
	}
	rt, err := ra.Mainloop.ks.AuthenticateWithCert(*ra.Mainloop.keygrant)
	if err != nil {
		return err
	}
	cert, err := keycommon.SendRequest(rt, ra.API, string(csr))
	if err != nil {
		return err
	}
	if len(cert) == 0 {
		return fmt.Errorf("Received empty response.")
	}
	// TODO: confirm it's valid before saving it?
	return ioutil.WriteFile(ra.CertFile, []byte(cert), os.FileMode(0644))
}
