package main

import (
	"os"
	"log"
	"io/ioutil"
	"errors"
	"keycommon"
	"encoding/json"
	"fmt"
	"crypto/tls"
	"keyclient"
)

// the keyclient is a daemon with a few different responsibilities:
//  - perform initial token authentication to get a keygranting certificate
//  - generate local key material
//  - renew the keygranting certificate
//  - renew other certificates

type configtype struct {
	AuthorityPath string
	Keyserver     string
	KeyPath       string
	CertPath      string
}

func main() {
	loop, err := keyclient.Load("/etc/hyades/keyclient/keyclient.conf")
	if err != nil {
		log.Fatal(err)
	}
	loop.Run()
}
