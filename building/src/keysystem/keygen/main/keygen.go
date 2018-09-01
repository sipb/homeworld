package main

import (
	"log"
	"os"
	"keysystem/keyserver/config"
	"keysystem/keygen"
)

func main() {
	logger := log.New(os.Stderr, "[keygen] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 4 {
		logger.Fatal("usage: keygen <keyserver-config> <authority-dir> <supervisor-group>\n  generates the authorities for a keyserver")
	}
	cfg, err := config.LoadRawConfig(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	authority_dir := os.Args[2]
	supervisor_group := os.Args[3]
	err = keygen.GenerateKeys(cfg, authority_dir, supervisor_group)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Print("done generating keys.")
}
