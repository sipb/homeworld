package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"keycommon/endpoint"
	"keycommon/reqtarget"
)

type authenticated struct {
	endpoint endpoint.ServerEndpoint
}

func (k *Keyserver) AuthenticateWithToken(token string) (reqtarget.RequestTarget, error) {
	if token == "" {
		return nil, errors.New("Invalid token.")
	}
	return &authenticated{k.endpoint.WithHeader("X-Bootstrap-Token", token)}, nil
}

func (k *Keyserver) AuthenticateWithCert(cert tls.Certificate) (reqtarget.RequestTarget, error) {
	return &authenticated{k.endpoint.WithCertificate(cert)}, nil
}

func (a *authenticated) SendRequests(reqs []reqtarget.Request) ([]string, error) {
	outputs := []string{}
	err := a.endpoint.PostJSON("/apirequest", reqs, &outputs)
	if err != nil {
		return nil, err
	}
	if len(outputs) != len(reqs) {
		return nil, fmt.Errorf("while finalizing response: wrong number of responses")
	}
	return outputs, nil
}
