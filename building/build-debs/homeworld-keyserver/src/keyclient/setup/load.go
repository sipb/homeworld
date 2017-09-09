package setup

import (
	"time"
	"keyclient/state"
	"keyclient/config"
	"keyclient/actions/bootstrap"
	"keyclient/actions/keygen"
	"keyclient/actions/keyreq"
	"keyclient/actions/download"
	"io/ioutil"
	"fmt"
	"keycommon/server"
	"keyclient/actloop"
	"log"
)

// TODO: private key rotation, not just getting new certs

func Load(configpath string, logger *log.Logger) ([]actloop.Action, error) {
	conf, err := config.LoadConfig(configpath)
	if err != nil {
		return nil, err
	}
	authoritydata, err := ioutil.ReadFile(conf.AuthorityPath)
	if err != nil {
		return nil, fmt.Errorf("while loading authority: %s", err)
	}
	ks, err := server.NewKeyserver(authoritydata, conf.Keyserver)
	if err != nil {
		return nil, fmt.Errorf("while preparing setup: %s", err)
	}
	s := &state.ClientState{Config: conf, Keyserver: ks}
	actions := []actloop.Action{}
	if conf.TokenPath != "" {
		act, err := bootstrap.PrepareBootstrapAction(s, conf.TokenPath, conf.TokenAPI)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, key := range conf.Keys {
		act, err := keygen.PrepareKeygenAction(key)
		if err != nil {
			return nil, err
		}
		if act != nil {
			actions = append(actions, act)
		}
		act, err = keyreq.PrepareRequestOrRenewKeys(s, key)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, dl := range conf.Downloads {
		act, err := download.PrepareDownloadAction(s, dl)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	err = s.ReloadKeygrantingCert()
	if err != nil {
		logger.Printf("keygranting cert not yet available: %s\n", err.Error())
	}
	return actions, nil
}

// TODO: unit-test this launch better (i.e. the ten second part, etc)
func Launch(actions []actloop.Action, logger *log.Logger) (stop func()) {
	loop := actloop.NewActLoop(actions, logger)
	go loop.Run(time.Second * 10)
	return loop.Cancel
}

func LoadAndLaunch(configpath string, logger *log.Logger) (stop func(), errout error) {
	actions, err := Load(configpath, logger)
	if err != nil {
		return nil, err
	}
	return Launch(actions, logger), nil
}
