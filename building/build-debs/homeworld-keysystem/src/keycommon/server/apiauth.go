package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"keycommon/endpoint"
	"keycommon/reqtarget"
	"keycommon/knc"
	"net/url"
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

func (k *Keyserver) AuthenticateWithKerberosTickets() (reqtarget.RequestTarget, error) {
	url, err := url.Parse(k.endpoint.BaseURL())
	if err != nil {
		return nil, err
	}

	return &knc.KncServer{url.Hostname()}, nil
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
