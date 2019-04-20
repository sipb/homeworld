package setup

import (
	"github.com/pkg/errors"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
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
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

func writeVariantNotice(variant string) error {
	err := ioutil.WriteFile("/etc/homeworld/config/keyserver.variant", []byte(variant+"\n"), 0644)
	if err != nil {
		return errors.Wrap(err, "while saving variant notice")
	}
	return nil
}

// TODO: private key rotation, not just getting new certs

func LoadDefault(logger *log.Logger) ([]actloop.Action, error) {
	variant, err := worldconfig.GetVariant()
	if err != nil {
		return nil, errors.Wrap(err, "determining variant")
	}
	err = writeVariantNotice(variant)
	if err != nil {
		return nil, err
	}
	ks, err := server.NewKeyserverDefault()
	if err != nil {
		return nil, errors.Wrap(err, "while preparing setup")
	}

	downloads := worldconfig.GetDownloads(variant)
	keys := worldconfig.GetKeys(variant)
	s := &state.ClientState{Keyserver: ks}

	// bootstrap actions
	act, err := keygen.PrepareKeygenAction(config.ConfigKey{Type: "tls", Key: paths.GrantingKeyPath})
	if err != nil {
		return nil, errors.Wrap(err, "while preparing keygen")
	}
	if act == nil {
		return nil, errors.New("expected non-nil result from PrepareKeygenAction")
	}
	act2, err := bootstrap.PrepareBootstrapAction(s, paths.BootstrapTokenPath, paths.BootstrapTokenAPI)
	if err != nil {
		return nil, err
	}
	actions := []actloop.Action{
		act,
		act2,
	}

	for _, key := range keys {
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
	for _, dl := range downloads {
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

func LoadAndLaunchDefault(logger *log.Logger) (stop func(), errout error) {
	actions, err := LoadDefault(logger)
	if err != nil {
		return nil, err
	}
	return Launch(actions, logger), nil
}
