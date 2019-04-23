package bootstrap

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type BootstrapAction struct {
	State         *state.ClientState
	TokenFilePath string
	TokenAPI      string
}

func (da *BootstrapAction) Info() string {
	return fmt.Sprintf("bootstrap with token API %s from path %s", da.TokenAPI, da.TokenFilePath)
}

func (da *BootstrapAction) getToken() (string, error) {
	contents, err := ioutil.ReadFile(da.TokenFilePath)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(contents))
	if len(token) == 0 {
		return "", errors.New("blank token found")
	}
	for _, c := range token {
		if !unicode.IsPrint(c) || c == ' ' || c > 127 {
			return "", fmt.Errorf("invalid token found: bad character '%c'", c)
		}
	}
	return token, nil
}

func (da *BootstrapAction) Pending() (bool, error) {
	return da.State.CanRetry(da.TokenAPI) && da.State.Keygrant == nil && fileutil.Exists(da.TokenFilePath), nil
}

func (da *BootstrapAction) CheckBlocker() error {
	if !fileutil.Exists(paths.GrantingKeyPath) {
		return fmt.Errorf("key does not yet exist: %s", paths.GrantingKeyPath)
	}
	return nil
}

func (da *BootstrapAction) buildCSR() ([]byte, error) {
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

func (da *BootstrapAction) sendRequest(api string, param string) (string, error) {
	token, err := da.getToken()
	if err != nil {
		return "", err
	}
	rt, err := da.State.Keyserver.AuthenticateWithToken(token) // can't fail, because getToken never returns an empty string
	if err != nil {
		return "", err
	}
	result, err := reqtarget.SendRequest(rt, api, param)
	if err != nil {
		if _, is := errors.Cause(err).(endpoint.OperationForbidden); is {
			da.State.RetryFailed(api)
		}
		return "", err
	}
	// remove token file, because it can't be used more than once, enforced by the server
	err = os.Remove(da.TokenFilePath)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (da *BootstrapAction) Perform(logger *log.Logger) error {
	csr, err := da.buildCSR()
	if err != nil {
		return err
	}
	certbytes, err := da.sendRequest(da.TokenAPI, string(csr))
	if err != nil {
		return err
	}
	if len(certbytes) == 0 {
		return errors.New("received empty response")
	}
	err = da.State.ReplaceKeygrantingCert([]byte(certbytes))
	if err != nil {
		return err
	}
	return nil
}
