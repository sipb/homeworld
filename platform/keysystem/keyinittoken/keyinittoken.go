package main

import (
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actions/keygen"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stderr, "[keyinittoken] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	err := keygen.PerformGenerate(paths.GrantingKeyPath)
	if err != nil {
		logger.Fatal(err)
	}
	fp, err := admit.LoadFingerprint(paths.GrantingKeyPath)
	if err != nil {
		logger.Fatal(err)
	}
	_, err = os.Stdout.WriteString(fp)
	if err != nil {
		logger.Fatal(err)
	}
}
