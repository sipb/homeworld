package keyreq

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"time"
	"util/wraputil"
	"keycommon/reqtarget"
	"util/csrutil"
	"keyclient/state"
	"errors"
	"keyclient/config"
	"keyclient/actloop"
	"log"
)

type RequestOrRenewAction struct {
	Mainloop        *state.ClientState
	InAdvance       time.Duration
	API             string
	Name            string
	CheckExpiration func([]byte) (time.Time, error)
	GenCSR          func([]byte) ([]byte, error)
	KeyFile         string
	CertFile        string
}

func PrepareRequestOrRenewKeys(m *state.ClientState, key config.ConfigKey, inadvance time.Duration) (actloop.Action, error) {
	if inadvance <= 0 {
		return nil, errors.New("Invalid in-advance for key renewal.")
	}
	if key.API == "" {
		return nil, errors.New("No renew API provided.")
	}
	switch key.Type {
	case "tls":
		fallthrough
	case "tls-pubkey":
		return &RequestOrRenewAction{Mainloop: m, InAdvance: inadvance, API: key.API, Name: key.Name, CheckExpiration: GetTLSCertExpiration, GenCSR: csrutil.BuildTLSCSR, KeyFile: key.Key, CertFile: key.Cert}, nil
	case "ssh":
		fallthrough
	case "ssh-pubkey":
		return &RequestOrRenewAction{Mainloop: m, InAdvance: inadvance, API: key.API, Name: key.Name, CheckExpiration: CheckSSHCertExpiration, GenCSR: csrutil.BuildSSHCSR, KeyFile: key.Key, CertFile: key.Cert}, nil
	default:
		return nil, fmt.Errorf("Unrecognized key type: %s", key.Type)
	}
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

func GetTLSCertExpiration(certdata []byte) (time.Time, error) {
	cert, err := wraputil.LoadX509CertFromPEM(certdata)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

func (ra *RequestOrRenewAction) Pending() (bool, error) {
	existing, err := ioutil.ReadFile(ra.CertFile)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // population needed
		} else {
			// this will probably fail to regenerate, but at least we tried? and this way, it's made clear that a problem is continuing.
			return true, fmt.Errorf("While trying to check expiration status of certificate: %s\n", err)
		}
	} else {
		expiration, err := ra.CheckExpiration(existing)
		if err != nil {
			// almost invariably means malformed
			return true, fmt.Errorf("While trying to check expiration status of certificate: %s\n", err)
		}
		renew_at := expiration.Add(-ra.InAdvance)
		if renew_at.After(time.Now()) {
			return false, nil // not time to renew
		} else {
			return true, nil // time to renew
		}
	}
}

func (ra *RequestOrRenewAction) CheckBlocker() error {
	if ra.Mainloop.Keygrant == nil {
		return errors.New("no keygranting certificate ready")
	} else {
		return nil
	}
}

func (ra *RequestOrRenewAction) Perform(logger *log.Logger) error {
	keydata, err := ioutil.ReadFile(ra.KeyFile)
	if err != nil {
		return err
	}
	csr, err := ra.GenCSR(keydata)
	if err != nil {
		return err
	}
	rt, err := ra.Mainloop.Keyserver.AuthenticateWithCert(*ra.Mainloop.Keygrant)
	if err != nil {
		return err
	}
	cert, err := reqtarget.SendRequest(rt, ra.API, string(csr))
	if err != nil {
		return err
	}
	if len(cert) == 0 {
		return fmt.Errorf("Received empty response.")
	}
	// TODO: confirm it's valid before saving it?
	return ioutil.WriteFile(ra.CertFile, []byte(cert), os.FileMode(0644))
}
