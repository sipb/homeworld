package verifier

import (
	"errors"
	"net/http"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/token"
)

const TokenHeader = "X-Bootstrap-Token"

type TokenVerifier struct {
	Registry *token.TokenRegistry
}

func NewTokenVerifier() TokenVerifier {
	reg := token.NewTokenRegistry()
	return TokenVerifier{reg}
}

func (v TokenVerifier) HasAttempt(request *http.Request) bool {
	return request.Header.Get(TokenHeader) != ""
}

func (v TokenVerifier) Verify(request *http.Request) (principal string, err error) {
	tokens := request.Header.Get(TokenHeader)
	if tokens == "" {
		return "", errors.New("No token authentication header provided")
	}
	tok, err := v.Registry.LookupToken(tokens)
	if err != nil {
		return "", err
	}
	err = tok.Claim()
	if err != nil {
		return "", err
	}
	return tok.Subject, nil
}
