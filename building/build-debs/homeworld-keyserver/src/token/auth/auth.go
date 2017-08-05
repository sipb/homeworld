package auth

import (
	"net/http"
	"errors"
	"token"
)

const authheader = "X-Bootstrap-Token"

func HasTokenAuthHeader(request *http.Request) bool {
	return request.Header.Get(authheader) != ""
}

func Authenticate(registry *token.TokenRegistry, request *http.Request) (string, error) {
	tokens := request.Header.Get(authheader)
	if tokens == "" {
		return "", errors.New("No token authentication header provided")
	}
	tok, err := registry.LookupToken(tokens)
	if err != nil {
		return "", err
	}
	err = tok.Claim()
	if err != nil {
		return "", err
	}
	return tok.Subject, nil
}
