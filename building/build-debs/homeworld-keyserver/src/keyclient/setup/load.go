package setup

import (
	"time"
	"keyclient/loop"
	"keyclient/config"
	"keyclient/actions/bootstrap"
	"keyclient/actions/keygen"
	"keyclient/actions/keyreq"
	"keyclient/actions/download"
	"io/ioutil"
	"fmt"
	"keycommon/server"
)

// TODO: private key rotation, not just getting new certs

func Load(configpath string) (*loop.Mainloop, error) {
	conf, err := config.LoadConfig(configpath)
	if err != nil {
		return nil, err
	}
	authoritydata, err := ioutil.ReadFile(conf.AuthorityPath)
	if err != nil {
		return nil, fmt.Errorf("While loading authority: %s", err)
	}
	ks, err := server.NewKeyserver(authoritydata, conf.Keyserver)
	if err != nil {
		return nil, fmt.Errorf("While preparing setup: %s", err)
	}
	m := &loop.Mainloop{Config: conf, Keyserver: ks}
	actions := []loop.Action{}
	if conf.TokenPath != "" {
		act, err := bootstrap.PrepareBootstrapAction(m, conf.TokenPath, conf.TokenAPI)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, key := range conf.Keys {
		inadvance, err := time.ParseDuration(key.InAdvance)
		if err != nil {
			return nil, err
		}
		act, err := keygen.PrepareKeygenAction(m, key)
		if err != nil {
			return nil, err
		}
		if act != nil {
			actions = append(actions, act)
		}
		act, err = keyreq.PrepareRequestOrRenewKeys(m, key, inadvance)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, dl := range conf.Downloads {
		act, err := download.PrepareDownloadAction(m, dl)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	m.Actions = actions
	m.ReloadKeygrantingCert()
	return m, nil
}