#!/bin/bash

cd "$(dirname "$0")/.."
GOPATH="$GOPATH:$(pwd)" go build src/integrationtesting/integrationhelper.go
GOPATH="$GOPATH:$(pwd)" go build src/keyserver/main/keyserver.go
GOPATH="$GOPATH:$(pwd)" go build src/keyclient/main/keyclient.go
