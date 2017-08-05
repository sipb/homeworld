package main

import (
	"keyserver"
	"log"
)

func main() {
	log.Fatal(keyserver.Run("/etc/hyades/keyserver/keyserver.conf"))
}
