#!/bin/bash

cd "$(dirname "$0")/.."
GOPATH="$GOPATH:$(pwd)" go build src/integrationtesting/integrationhelper.go
GOPATH="$GOPATH:$(pwd)" go build src/keyserver.go
GOPATH="$GOPATH:$(pwd)" go build src/keyclient.go
