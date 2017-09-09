package bootstrap

import (
	"errors"
	"fmt"
	"io/ioutil"
	"keyclient/actloop"
	"keyclient/state"
	"keycommon/reqtarget"
	"log"
	"os"
	"strings"
	"unicode"
	"util/csrutil"
	"util/fileutil"
)

type BootstrapAction struct {
	State         *state.ClientState
	TokenFilePath string
	TokenAPI      string
}

func PrepareBootstrapAction(s *state.ClientState, tokenfilepath string, api string) (actloop.Action, error) {
	if api == "" {
		return nil, errors.New("no bootstrap api provided")
	}
	return &BootstrapAction{State: s, TokenFilePath: tokenfilepath, TokenAPI: api}, nil
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
	return da.State.Keygrant == nil && fileutil.Exists(da.TokenFilePath), nil
}

func (da *BootstrapAction) CheckBlocker() error {
	return nil
}

func (da *BootstrapAction) buildCSR() ([]byte, error) {
	privkey, err := ioutil.ReadFile(da.State.Config.KeyPath)
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
		return fmt.Errorf("received empty response")
	}
	err = da.State.ReplaceKeygrantingCert([]byte(certbytes))
	if err != nil {
		return err
	}
	return nil
}
