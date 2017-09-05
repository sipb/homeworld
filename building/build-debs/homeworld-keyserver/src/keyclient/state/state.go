package state

import (
	"crypto/tls"
	"util/fileutil"
	"keycommon/server"
	"keyclient/config"
	"fmt"
	"errors"
)

type ClientState struct {
	Keyserver   *server.Keyserver
	Config      config.Config
	Keygrant    *tls.Certificate
}

func (m *ClientState) ReloadKeygrantingCert() error {
	if fileutil.Exists(m.Config.KeyPath) && fileutil.Exists(m.Config.CertPath) {
		cert, err := tls.LoadX509KeyPair(m.Config.CertPath, m.Config.KeyPath)
		if err != nil {
			return fmt.Errorf("Failed to reload keygranting certificate: %s", err)
		} else {
			m.Keygrant = &cert
			return nil
		}
	} else {
		return errors.New("No keygranting certificate found.")
	}
}
