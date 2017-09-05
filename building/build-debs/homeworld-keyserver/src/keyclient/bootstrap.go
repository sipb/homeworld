package keyclient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
	"keycommon/csr"
	"keycommon/reqtarget"
)

type BootstrapAction struct {
	Mainloop      *Mainloop
	TokenFilePath string
	TokenAPI      string
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
	if da.Mainloop.keygrant != nil {
		return ErrNothingToDo
	}
	token, err := da.getToken()
	if err != nil {
		return err
	}
	if token == "" {
		return ErrNothingToDo
	}
	rt, err := da.Mainloop.ks.AuthenticateWithToken(token)
	if err != nil {
		return err
	}
	privkey, err := ioutil.ReadFile(da.Mainloop.config.KeyPath)
	if err != nil {
		return err
	}
	csr, err := csr.BuildTLSCSR(privkey)
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
	err = ioutil.WriteFile(da.Mainloop.config.CertPath, []byte(certbytes), os.FileMode(0600))
	if err != nil {
		return err
	}
	da.Mainloop.reloadKeygrantingCert()
	if da.Mainloop.keygrant == nil {
		return fmt.Errorf("Expected properly loaded keygrant certificate")
	}
	return nil
}
