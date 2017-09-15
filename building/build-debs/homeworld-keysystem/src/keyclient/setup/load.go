package setup

import (
	"fmt"
	"io/ioutil"
	"keyclient/actions/bootstrap"
	"keyclient/actions/download"
	"keyclient/actions/keygen"
	"keyclient/actions/keyreq"
	"keyclient/actloop"
	"keyclient/config"
	"keyclient/state"
	"keycommon/server"
	"log"
	"time"
	"errors"
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
		// for generating private keys
		act, err := keygen.PrepareKeygenAction(config.ConfigKey{Type: "tls", Key: conf.KeyPath})
		if err != nil {
			return nil, err
		}
		if act == nil {
			return nil, errors.New("expected non-nil result from PrepareKeygenAction")
		}
		actions = append(actions, act)
		// for bootstrapping
		act, err = bootstrap.PrepareBootstrapAction(s, conf.TokenPath, conf.TokenAPI)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, key := range conf.Keys {
		// for generating private keys
		act, err := keygen.PrepareKeygenAction(key)
		if err != nil {
			return nil, err
		}
		if act != nil {
			actions = append(actions, act)
		}
		// for getting certificates for keys
		act, err = keyreq.PrepareRequestOrRenewKeys(s, key)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, dl := range conf.Downloads {
		// for downloading files and public keys
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
	go loop.Run(time.Second * 2, time.Minute * 5)
	return loop.Cancel
}

func LoadAndLaunch(configpath string, logger *log.Logger) (stop func(), errout error) {
	actions, err := Load(configpath, logger)
	if err != nil {
		return nil, err
	}
	return Launch(actions, logger), nil
}
