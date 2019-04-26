package setup

import (
	"github.com/pkg/errors"
	"log"
	"os/exec"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

// TODO: private key rotation, not just getting new certs

func LoadDefault(logger *log.Logger) (actloop.NewAction, error) {
	ks, err := server.NewKeyserverDefault()
	if err != nil {
		return nil, errors.Wrap(err, "while preparing setup")
	}

	s := state.NewClientState(ks)

	actions, err := worldconfig.BuildActions(s)
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
func Launch(actions actloop.NewAction, logger *log.Logger) (stop func()) {
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
