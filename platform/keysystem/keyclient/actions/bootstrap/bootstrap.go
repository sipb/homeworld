package bootstrap

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"strings"
	"unicode"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

func Bootstrap(api string, nac *actloop.NewActionContext) {
	if !nac.State.CanRetry(api) {
		// nothing to do
	} else if nac.State.Keygrant != nil {
		// nothing to do
	} else if !fileutil.Exists(paths.BootstrapTokenPath) {
		// nothing to do
	} else if !fileutil.Exists(paths.GrantingKeyPath) {
		nac.Blocked(fmt.Errorf("key does not yet exist: %s", paths.GrantingKeyPath))
	} else {
		info := fmt.Sprintf("bootstrap with token API %s from path %s", api, paths.BootstrapTokenPath)
		err := bootstrap(api, nac.State)
		if err != nil {
			nac.Errored(info, err)
		} else {
			nac.NotifyPerformed(info)
		}
	}
}

func getToken() (string, error) {
	contents, err := ioutil.ReadFile(paths.BootstrapTokenPath)
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

func sendRequest(state *state.ClientState, api string, param string) (string, error) {
	token, err := getToken()
	if err != nil {
		return "", err
	}
	rt, err := state.Keyserver.AuthenticateWithToken(token) // can't fail, because getToken never returns an empty string
	if err != nil {
		return "", err
	}
	result, err := reqtarget.SendRequest(rt, api, param)
	if err != nil {
		if _, is := errors.Cause(err).(endpoint.OperationForbidden); is {
			state.RetryFailed(api)
		}
		return "", err
	}
	// remove token file, because it can't be used more than once, enforced by the server
	err = os.Remove(paths.BootstrapTokenPath)
	if err != nil {
		return "", err
	}
	return result, nil
}

func bootstrap(api string, state *state.ClientState) error {
	csr, err := buildCSR()
	if err != nil {
		return err
	}
	certbytes, err := sendRequest(state, api, string(csr))
	if err != nil {
		return err
	}
	if len(certbytes) == 0 {
		return errors.New("received empty response")
	}
	err = state.ReplaceKeygrantingCert([]byte(certbytes))
	if err != nil {
		return err
	}
	return nil
}
