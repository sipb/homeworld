package main

import (
	"os"
	"log"
	"io/ioutil"
	"errors"
	"keycommon"
	"encoding/json"
)

func HandleRequest(principal string, request_data []byte) ([]byte, error) {
	jsonload := []struct{ api string; body string } {}
	err := json.Unmarshal(request_data, jsonload)
	if err != nil {
		return nil, err
	}

	requests := make([]keycommon.Request, len(jsonload))
	for i, req := range jsonload {
		requests[i].API = req.api
		requests[i].Body = req.body
	}

	_, rt, err := keycommon.LoadKeyserverWithCert("/etc/hyades/keyserver/keyclient.conf")
	if err != nil {
		return nil, err
	}

	reqt, err := keycommon.Impersonate(rt, "auth-to-kerberos", principal)
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

func Process() error {
	if os.Getenv("KNC_MECH") != "krb5" {
		return errors.New("Expected kerberos authentication.")
	}

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

	result, err := HandleRequest(kncCreds, request_data)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(result)
	return err
}

func main() {
	err := Process()
	// TODO: verify that stderr does *not* get sent across knc
	if err != nil {
		log.Fatal(err)
	}
}
