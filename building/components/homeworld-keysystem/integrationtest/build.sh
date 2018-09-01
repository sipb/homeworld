#!/bin/bash

cd "$(dirname "$0")/.."
GOPATH="$GOPATH:$(pwd)" go build src/keysystem/integrationtesting/integrationhelper.go
GOPATH="$GOPATH:$(pwd)" go build src/keysystem/keyserver/main/keyserver.go
GOPATH="$GOPATH:$(pwd)" go build src/keysystem/keyclient/main/keyclient.go
