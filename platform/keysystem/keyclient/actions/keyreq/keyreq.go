package keyreq

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type RequestOrRenewAction struct {
	State           *state.ClientState
	InAdvance       time.Duration
	API             string
	CheckExpiration func([]byte) (time.Time, error)
	GenCSR          func([]byte) ([]byte, error)
	KeyFile         string
	CertFile        string
}

func (ra *RequestOrRenewAction) Info() string {
	return fmt.Sprintf("req/renew key %s into cert %s with API %s in advance by %v", ra.CertFile, ra.KeyFile, ra.InAdvance, ra.API)
}

func (ra *RequestOrRenewAction) Pending() (bool, error) {
	if !ra.State.CanRetry(ra.API) {
		return false, nil
	}
	existing, err := ioutil.ReadFile(ra.CertFile)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // population needed
		} else {
			// this will probably fail to regenerate, but at least we tried? and this way, it's made clear that a problem is continuing.
			return true, errors.Wrap(err, "while trying to check expiration status of certificate")
		}
	} else {
		expiration, err := ra.CheckExpiration(existing)
		if err != nil {
			// almost invariably means malformed
			return true, errors.Wrap(err, "while trying to check expiration status of certificate")
		}
		renewAt := expiration.Add(-ra.InAdvance)
		if renewAt.After(time.Now()) {
			return false, nil // not time to renew
		} else {
			return true, nil // time to renew
		}
	}
}

func (ra *RequestOrRenewAction) CheckBlocker() error {
	if ra.State.Keygrant == nil {
		return errors.New("no keygranting certificate ready")
	} else if !fileutil.Exists(ra.KeyFile) {
		return fmt.Errorf("key does not yet exist: %s", ra.KeyFile)
	} else {
		return nil
	}
}

func (ra *RequestOrRenewAction) Perform(_ *log.Logger) error {
	keydata, err := ioutil.ReadFile(ra.KeyFile)
	if err != nil {
		return errors.Wrap(err, "while reading keyfile")
	}
	csr, err := ra.GenCSR(keydata)
	if err != nil {
		return errors.Wrap(err, "while generating CSR")
	}
	rt, err := ra.State.Keyserver.AuthenticateWithCert(*ra.State.Keygrant)
	if err != nil {
		return errors.Wrap(err, "while authenticating with cert") // no actual way for this to fail
	}
	cert, err := reqtarget.SendRequest(rt, ra.API, string(csr))
	if err != nil {
		if _, is := err.(endpoint.OperationForbidden); is {
			ra.State.RetryFailed(ra.API)
		}
		return errors.Wrap(err, "while sending request")
	}
	if len(cert) == 0 {
		return errors.New("while sending request: received empty response")
	}
	// TODO: confirm it's valid before saving it?
	err = ioutil.WriteFile(ra.CertFile, []byte(cert), os.FileMode(0644))
	if err != nil {
		return errors.Wrap(err, "while writing result")
	}
	certabs, err := filepath.Abs(ra.CertFile)
	if err != nil {
		return err
	}
	grantabs, err := filepath.Abs(paths.GrantingCertPath)
	if err != nil {
		return err
	}
	if certabs == grantabs {
		err := ra.State.ReloadKeygrantingCert()
		if err != nil {
			return err
		}
	}
	return nil
}
