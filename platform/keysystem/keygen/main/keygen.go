package main

import (
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
)

func main() {
	logger := log.New(os.Stderr, "[keygen] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 3 {
		logger.Fatal("usage: keygen <keyserver-config> <authority-dir>\n  generates the authorities for a keyserver")
	}
	cfg, err := config.LoadRawConfig(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	authorityDir := os.Args[2]
	err = keygen.GenerateKeys(cfg, authorityDir)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Print("done generating keys.")
}
