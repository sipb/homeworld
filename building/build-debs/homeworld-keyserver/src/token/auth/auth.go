package auth

import (
	"net/http"
	"errors"
	"util"
	"token"
)

const authheader = "X-Bootstrap-Token"

func HasTokenAuthHeader(request *http.Request) bool {
	return request.Header.Get(authheader) != ""
}

func Authenticate(registry *token.TokenRegistry, request *http.Request) (string, error) {
	token := request.Header.Get(authheader)
	if token == "" {
		return "", errors.New("No token authentication header provided")
	}
	tok, err := registry.LookupToken(token)
	if err != nil {
		return "", err
	}
	ip, err := util.ParseRemoteAddressFromRequest(request)
	if err != nil {
		return "", err
	}
	err = tok.Claim(ip)
	if err != nil {
		return "", err
	}
	return tok.Subject, nil
}
