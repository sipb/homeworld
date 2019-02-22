package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/api"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
)

func HandleRequest(principal string, request_data []byte, configfile string) ([]byte, error) {
	requests := []reqtarget.Request{}
	err := json.Unmarshal(request_data, &requests)
	if err != nil {
		return nil, err
	}

	_, rt, err := api.LoadKeyserverWithCert(configfile)
	if err != nil {
		return nil, err
	}

	reqt, err := reqtarget.Impersonate(rt, "auth-to-kerberos", principal)
	if err != nil {
		return nil, err
	}

	result, err := reqt.SendRequests(requests)
	if err != nil {
		return nil, err
	}

	if len(result) != len(requests) {
		return nil, errors.New("Wrong number of results")
	}

	return json.Marshal(result)
}

func Process(configfile string) error {
	kncCreds := os.Getenv("KNC_CREDS")

	if kncCreds == "" {
		return errors.New("No credentials supplied!")
	}

	request_data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if len(request_data) == 0 {
		return errors.New("Empty request.")
	}

	result, err := HandleRequest(kncCreds, request_data, configfile)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(result)
	return err
}

func main() {
	logger := log.New(os.Stderr, "[keygateway] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 2 {
		logger.Fatal("no configuration file provided")
	}
	err := Process(os.Args[1])
	// TODO: verify that stderr does *not* get sent across knc
	if err != nil {
		logger.Fatal(err)
	}
}
