package bootstrap

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"util/csrutil"
	"keycommon/reqtarget"
	"keyclient/loop"
)

type BootstrapAction struct {
	Mainloop      *loop.Mainloop
	TokenFilePath string
	TokenAPI      string
}

func PrepareBootstrapAction(m *loop.Mainloop, tokenfilepath string, api string) (loop.Action, error) {
	if api == "" {
		return nil, errors.New("No bootstrap API provided.")
	}
	return &BootstrapAction{Mainloop: m, TokenFilePath: tokenfilepath, TokenAPI: api}, nil
}

func (da *BootstrapAction) getToken() (string, error) {
	contents, err := ioutil.ReadFile(da.TokenFilePath)
	if os.IsNotExist(err) {
		return "", nil
	}
	token := strings.TrimSpace(string(contents))
	for _, c := range token {
		if !unicode.IsPrint(c) || c == ' ' || c > 127 {
			return "", errors.New("Invalid token found.")
		}
	}
	return token, nil
}

func (da *BootstrapAction) Perform() error {
	if da.Mainloop.Keygrant != nil {
		return loop.ErrNothingToDo
	}
	token, err := da.getToken()
	if err != nil {
		return err
	}
	if token == "" {
		return loop.ErrNothingToDo
	}
	rt, err := da.Mainloop.Keyserver.AuthenticateWithToken(token)
	if err != nil {
		return err
	}
	privkey, err := ioutil.ReadFile(da.Mainloop.Config.KeyPath)
	if err != nil {
		return err
	}
	csr, err := csrutil.BuildTLSCSR(privkey)
	if err != nil {
		return err
	}
	certbytes, err := reqtarget.SendRequest(rt, da.TokenAPI, string(csr))
	if err != nil {
		return err
	}
	// remove token file, because it can't be used more than once, enforced by the server
	err = os.Remove(da.TokenFilePath)
	if err != nil {
		return err
	}
	if len(certbytes) == 0 {
		return fmt.Errorf("Received empty response.")
	}
	err = ioutil.WriteFile(da.Mainloop.Config.CertPath, []byte(certbytes), os.FileMode(0600))
	if err != nil {
		return err
	}
	da.Mainloop.ReloadKeygrantingCert()
	if da.Mainloop.Keygrant == nil {
		return fmt.Errorf("Expected properly loaded keygrant certificate")
	}
	return nil
}
