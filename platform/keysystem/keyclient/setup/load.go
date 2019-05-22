package setup

import (
	"github.com/pkg/errors"
	"log"
	"os/exec"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
)

// TODO: private key rotation, not just getting new certs

func notifyReady(logger *log.Logger) {
	// tells systemd that we're done setting up
	err := exec.Command("systemd-notify", "--ready").Run()
	if err != nil {
		logger.Printf("failed to notify systemd of readiness: %v\n", err)
	}
}

func LoadAndLaunchDefault(actions actloop.NewAction, logger *log.Logger) error {
	ks, err := api.LoadDefaultKeyserver()
	if err != nil {
		return errors.Wrap(err, "while preparing setup")
	}

	clientState, warning := state.NewClientState(ks)
	if warning != nil {
		logger.Println(warning)
	}

	loop := actloop.NewActLoop(actions, logger)
	go loop.Run(clientState, time.Second*2, time.Minute*5, notifyReady)
	return nil
}
