package setup

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
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

	s := &state.ClientState{Keyserver: ks}

	actions, err := worldconfig.BuildActions(s, variant)
	if err != nil {
		return nil, err
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
