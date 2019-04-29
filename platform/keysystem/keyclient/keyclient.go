package main

import (
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/setup"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

// the keyclient is a daemon with a few different responsibilities:
//  - perform initial token authentication to get a keygranting certificate
//  - generate local key material
//  - renew the keygranting certificate
//  - renew other certificates

func main() {
	logger := log.New(os.Stderr, "[keyclient] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	err := setup.LoadAndLaunchDefault(worldconfig.ConvergeState, logger)
	if err != nil {
		logger.Fatal(err)
	}
	// hang forever
	<-make(chan int)
}
