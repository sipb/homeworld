package keyreq

import (
	"errors"
	"fmt"
	"io/ioutil"
	"keyclient/actloop"
	"keyclient/config"
	"keyclient/state"
	"keycommon/reqtarget"
	"log"
	"os"
	"time"
	"util/certutil"
	"util/csrutil"
)

type RequestOrRenewAction struct {
	State           *state.ClientState
	InAdvance       time.Duration
	API             string
	Name            string
	CheckExpiration func([]byte) (time.Time, error)
	GenCSR          func([]byte) ([]byte, error)
	KeyFile         string
	CertFile        string
}

func PrepareRequestOrRenewKeys(s *state.ClientState, key config.ConfigKey) (actloop.Action, error) {
	inadvance, err := time.ParseDuration(key.InAdvance)
	if err != nil {
		return nil, fmt.Errorf("invalid in-advance interval for key renewal: %s", err.Error())
	}
	if inadvance <= 0 {
		return nil, errors.New("invalid in-advance interval for key renewal: nonpositive duration")
	}
	if key.API == "" {
		return nil, errors.New("no renew api provided")
	}
	action := &RequestOrRenewAction{
		State:     s,
		InAdvance: inadvance,
		API:       key.API,
		Name:      key.Name,
		KeyFile:   key.Key,
		CertFile:  key.Cert,
	}
	switch key.Type {
	case "tls":
		action.CheckExpiration = certutil.CheckTLSCertExpiration
		action.GenCSR = csrutil.BuildTLSCSR
		return action, nil
	case "ssh":
		fallthrough
	case "ssh-pubkey":
		action.CheckExpiration = certutil.CheckSSHCertExpiration
		action.GenCSR = csrutil.BuildSSHCSR
		return action, nil
	default:
		return nil, fmt.Errorf("unrecognized key type: %s", key.Type)
	}
}

func (ra *RequestOrRenewAction) Pending() (bool, error) {
	existing, err := ioutil.ReadFile(ra.CertFile)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // population needed
		} else {
			// this will probably fail to regenerate, but at least we tried? and this way, it's made clear that a problem is continuing.
			return true, fmt.Errorf("while trying to check expiration status of certificate: %s", err)
		}
	} else {
		expiration, err := ra.CheckExpiration(existing)
		if err != nil {
			// almost invariably means malformed
			return true, fmt.Errorf("while trying to check expiration status of certificate: %s", err)
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
	if ra.State.Keygrant == nil {
		return errors.New("no keygranting certificate ready")
	} else {
		return nil
	}
}

func (ra *RequestOrRenewAction) Perform(_ *log.Logger) error {
	keydata, err := ioutil.ReadFile(ra.KeyFile)
	if err != nil {
		return fmt.Errorf("while reading keyfile: %s", err.Error())
	}
	csr, err := ra.GenCSR(keydata)
	if err != nil {
		return fmt.Errorf("while generating CSR: %s", err.Error())
	}
	rt, err := ra.State.Keyserver.AuthenticateWithCert(*ra.State.Keygrant)
	if err != nil {
		return fmt.Errorf("while authenticating with cert: %s", err.Error()) // no actual way for this to fail
	}
	cert, err := reqtarget.SendRequest(rt, ra.API, string(csr))
	if err != nil {
		return fmt.Errorf("while sending request: %s", err.Error())
	}
	if len(cert) == 0 {
		return fmt.Errorf("while sending request: received empty response")
	}
	// TODO: confirm it's valid before saving it?
	err = ioutil.WriteFile(ra.CertFile, []byte(cert), os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("while writing result: %s", err.Error())
	}
	return nil
}
