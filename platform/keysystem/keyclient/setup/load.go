package setup

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/bootstrap"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/download"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keyreq"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/config"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
)

// TODO: private key rotation, not just getting new certs

func Load(configpath string, logger *log.Logger) ([]actloop.Action, error) {
	conf, err := config.LoadConfig(configpath)
	if err != nil {
		return nil, err
	}
	authoritydata, err := ioutil.ReadFile(conf.AuthorityPath)
	if err != nil {
		return nil, errors.Wrap(err, "while loading authority")
	}
	ks, err := server.NewKeyserver(authoritydata, conf.Keyserver)
	if err != nil {
		return nil, errors.Wrap(err, "while preparing setup")
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

func notifyReady(logger *log.Logger) {
	// tells systemd that we're done setting up
	err := exec.Command("systemd-notify", "--ready").Run()
	if err != nil {
		logger.Printf("failed to notify systemd of readiness: %v\n", err)
	}
}

// TODO: unit-test this launch better (i.e. the ten second part, etc)
func Launch(actions []actloop.Action, logger *log.Logger) (stop func()) {
	loop := actloop.NewActLoop(actions, logger)
	go loop.Run(time.Second*2, time.Minute*5, notifyReady)
	return loop.Cancel
}

func LoadAndLaunch(configpath string, logger *log.Logger) (stop func(), errout error) {
	actions, err := Load(configpath, logger)
	if err != nil {
		return nil, err
	}
	return Launch(actions, logger), nil
}
