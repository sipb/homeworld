package main

import (
	"keysystem/keyserver/keyapi"
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stderr, "[keyserver] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 2 {
		logger.Fatal("no configuration file provided")
	}
	_, onstop, err := keyapi.Run(os.Args[1], ":20557", logger)
	if err != nil {
		logger.Fatal(err)
	} else {
		logger.Fatal(<-onstop)
	}
}
