package main

import (
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/keygen"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

func main() {
	logger := log.New(os.Stderr, "[keygen] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 3 {
		logger.Fatal("usage: keygen <keyserver-config> <authority-dir>\n  generates the authorities for a keyserver")
	}
	setup, err := worldconfig.LoadSpireSetup(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	authorityDir := os.Args[2]
	err = keygen.GenerateKeys(setup, authorityDir)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Print("done generating keys.")
}
