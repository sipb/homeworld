package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/keyapi"
)

func main() {
	logger := log.New(os.Stderr, "[keyserver] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 1 {
		logger.Fatalln("usage: keyserver")
	}
	_, onstop, err := keyapi.Run(":20557", logger)
	if err != nil {
		logger.Fatal(err)
	} else {
		err := exec.Command("systemd-notify", "--ready").Run()
		if err != nil {
			logger.Fatal("failed to notify systemd of readiness: %v\n", err)
		} else {
			// service is up, wait for kill signal
			logger.Fatal(<-onstop)
		}
	}
}
