package main

import (
	"keyclient"
	"log"
)

// the keyclient is a daemon with a few different responsibilities:
//  - perform initial token authentication to get a keygranting certificate
//  - generate local key material
//  - renew the keygranting certificate
//  - renew other certificates

func main() {
	loop, err := keyclient.Load("/etc/hyades/keyclient/keyclient.conf")
	if err != nil {
		log.Fatal(err)
	}
	loop.Run()
}
