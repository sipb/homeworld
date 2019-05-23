package bootstrap

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

func Bootstrap(nac *actloop.NewActionContext) {
	if nac.State.Keygrant != nil {
		// nothing to do
	} else if !fileutil.Exists(paths.GrantingKeyPath) {
		nac.Blocked(fmt.Errorf("key does not exist: %s", paths.GrantingKeyPath))
	} else {
		info := "requesting admission..."
		err := bootstrap(nac, info)
		if err != nil {
			nac.Errored(info, err)
		}
	}
}

func buildCSR() ([]byte, error) {
	privkey, err := ioutil.ReadFile(paths.GrantingKeyPath)
	if err != nil {
		return nil, err
	}
	csr, err := csrutil.BuildTLSCSR(privkey)
	if err != nil {
		return nil, err
	}
	return csr, err
}

func bootstrap(nac *actloop.NewActionContext, info string) error {
	csr, err := buildCSR()
	if err != nil {
		return err
	}
	// we expect this to repeatedly error out until approval
	certbytes, err := nac.State.Keyserver.RequestAdmission(csr)
	if err != nil {
		nac.Blocked(err)
		return nil
	}
	if len(certbytes) == 0 {
		return errors.New("received empty response")
	}
	err = nac.State.ReplaceKeygrantingCert([]byte(certbytes))
	if err != nil {
		return err
	}
	nac.NotifyPerformed(info)
	return nil
}
