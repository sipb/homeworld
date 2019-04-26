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

func LoadDefault(logger *log.Logger) (*state.ClientState, actloop.NewAction, error) {
	ks, err := server.NewKeyserverDefault()
	if err != nil {
		return nil, nil, errors.Wrap(err, "while preparing setup")
	}

	s := state.NewClientState(ks)

	actions := worldconfig.BuildActions(s)

	err = s.ReloadKeygrantingCert()
	if err != nil {
		logger.Printf("keygranting cert not yet available: %s\n", err.Error())
	}
	return s, actions, nil
}

func notifyReady(logger *log.Logger) {
	// tells systemd that we're done setting up
	err := exec.Command("systemd-notify", "--ready").Run()
	if err != nil {
		logger.Printf("failed to notify systemd of readiness: %v\n", err)
	}
}

// TODO: unit-test this launch better (i.e. the ten second part, etc)
func Launch(state *state.ClientState, actions actloop.NewAction, logger *log.Logger) (stop func()) {
	loop := actloop.NewActLoop(actions, logger)
	go loop.Run(state, time.Second*2, time.Minute*5, notifyReady)
	return loop.Cancel
}

func LoadAndLaunchDefault(logger *log.Logger) (stop func(), errout error) {
	state, actions, err := LoadDefault(logger)
	if err != nil {
		return nil, err
	}
	return Launch(state, actions, logger), nil
}
