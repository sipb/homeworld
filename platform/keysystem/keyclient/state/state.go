package state

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/config"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type ClientState struct {
	Keyserver *server.Keyserver
	Config    config.Config
	Keygrant  *tls.Certificate
}

func (s *ClientState) ReloadKeygrantingCert() error {
	if fileutil.Exists(s.Config.KeyPath) && fileutil.Exists(s.Config.CertPath) {
		cert, err := tls.LoadX509KeyPair(s.Config.CertPath, s.Config.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to reload keygranting certificate: %s", err)
		} else {
			s.Keygrant = &cert
			return nil
		}
	} else {
		return errors.New("no keygranting certificate found")
	}
}

func (s *ClientState) ReplaceKeygrantingCert(data []byte) error {
	err := fileutil.EnsureIsFolder(path.Dir(s.Config.CertPath)) // TODO: unit test
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.Config.CertPath, data, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = s.ReloadKeygrantingCert()
	if err != nil {
		return fmt.Errorf("expected properly loaded keygrant certificate, but: %s", err)
	}
	return nil
}
