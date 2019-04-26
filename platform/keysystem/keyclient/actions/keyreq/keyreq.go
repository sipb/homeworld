package keyreq

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/certutil"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type RequestOrRenewAction struct {
	InAdvance       time.Duration
	API             string
	CheckExpiration func([]byte) (time.Time, error)
	GenCSR          func([]byte) ([]byte, error)
	KeyFile         string
	CertFile        string
}

func RequestOrRenewTLSKey(key string, cert string, api string, inadvance time.Duration, nac *actloop.NewActionContext) {
	action := &RequestOrRenewAction{
		InAdvance:       inadvance,
		API:             api,
		KeyFile:         key,
		CertFile:        cert,
		CheckExpiration: certutil.CheckTLSCertExpiration,
		GenCSR:          csrutil.BuildTLSCSR,
	}
	action.Act(nac)
}

func RequestOrRenewSSHKey(key string, cert string, api string, inadvance time.Duration, nac *actloop.NewActionContext) {
	action := &RequestOrRenewAction{
		InAdvance:       inadvance,
		API:             api,
		KeyFile:         key,
		CertFile:        cert,
		CheckExpiration: certutil.CheckSSHCertExpiration,
		GenCSR:          csrutil.BuildSSHCSR,
	}
	action.Act(nac)
}

func (ra *RequestOrRenewAction) shouldRegenerate(nac *actloop.NewActionContext, info string) bool {
	existing, err := ioutil.ReadFile(ra.CertFile)
	if err != nil {
		if !os.IsNotExist(err) {
			// this will probably fail to regenerate, but at least we tried? and this way, it's made clear that a problem is continuing.
			nac.Errored(info, err)
		}
		// fix missing or broken certificate by renewal
		return true
	}
	expiration, err := ra.CheckExpiration(existing)
	if err != nil {
		nac.Errored(info, errors.Wrap(err, "while trying to check expiration status of certificate"))
		return true // fix malformed certificate by renewal
	}
	renewAt := expiration.Add(-ra.InAdvance)
	if renewAt.After(time.Now()) {
		return false // not time to renew
	} else {
		return true // time to renew
	}
}

// if we just renewed the keygranting certificate, reload it
func (ra *RequestOrRenewAction) checkReload(nac *actloop.NewActionContext) error {
	certabs, err := filepath.Abs(ra.CertFile)
	if err != nil {
		return err
	}
	grantabs, err := filepath.Abs(paths.GrantingCertPath)
	if err != nil {
		return err
	}
	if certabs == grantabs {
		err := nac.State.ReloadKeygrantingCert()
		if err != nil {
			return err
		}
	}
	return nil
}

func (ra *RequestOrRenewAction) generateCSR() ([]byte, error) {
	keydata, err := ioutil.ReadFile(ra.KeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "while reading keyfile")
	}
	csr, err := ra.GenCSR(keydata)
	if err != nil {
		return nil, errors.Wrap(err, "while generating CSR")
	}
	return csr, err
}

func (ra *RequestOrRenewAction) requestSignature(csr []byte, nac *actloop.NewActionContext) ([]byte, error) {
	rt, err := nac.State.Keyserver.AuthenticateWithCert(*nac.State.Keygrant)
	if err != nil {
		return nil, errors.Wrap(err, "while authenticating with cert") // no actual way for this to fail here
	}
	cert, err := reqtarget.SendRequest(rt, ra.API, string(csr))
	if err != nil {
		if _, is := err.(endpoint.OperationForbidden); is {
			nac.State.RetryFailed(ra.API)
		}
		return nil, errors.Wrap(err, "while sending request")
	}
	if len(cert) == 0 {
		return nil, errors.New("while sending request: received empty response")
	}
	return []byte(cert), nil
}

func (ra *RequestOrRenewAction) regenerate(nac *actloop.NewActionContext) error {
	csr, err := ra.generateCSR()
	if err != nil {
		return err
	}
	cert, err := ra.requestSignature(csr, nac)
	if err != nil {
		return err
	}
	// TODO: confirm it's valid before saving it?
	err = ioutil.WriteFile(ra.CertFile, cert, os.FileMode(0644))
	if err != nil {
		return errors.Wrap(err, "while writing result")
	}
	err = ra.checkReload(nac)
	if err != nil {
		return errors.Wrap(err, "while reloading granting cert")
	}
	return nil
}

func (ra *RequestOrRenewAction) Act(nac *actloop.NewActionContext) {
	info := fmt.Sprintf("req/renew key %s into cert %s with API %s in advance by %v", ra.CertFile, ra.KeyFile, ra.InAdvance, ra.API)
	if !nac.State.CanRetry(ra.API) {
		// nothing to do
	} else if !ra.shouldRegenerate(nac, info) {
		// nothing to do
	} else if nac.State.Keygrant == nil {
		nac.Blocked(errors.New("no keygranting certificate ready"))
	} else if !fileutil.Exists(ra.KeyFile) {
		nac.Blocked(fmt.Errorf("key does not yet exist: %s", ra.KeyFile))
	} else {
		err := ra.regenerate(nac)
		if err != nil {
			nac.Errored(info, err)
		} else {
			nac.NotifyPerformed(info)
		}
	}
}
