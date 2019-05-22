package main

import (
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/keygen"
)

func main() {
	logger := log.New(os.Stderr, "[keygen] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 2 {
		logger.Fatal("usage: keygen <authority-dir>\n  generates the authorities for a keyserver")
	}
	authorityDir := os.Args[1]
	err := keygen.GenerateKeys(authorityDir)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Print("done generating keys.")
}
