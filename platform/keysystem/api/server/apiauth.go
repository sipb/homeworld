package server

import (
	"crypto/tls"
	"errors"
	"net/url"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/knc"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
)

type authenticated struct {
	endpoint endpoint.ServerEndpoint
}

func (k *Keyserver) AuthenticateWithCert(cert tls.Certificate) (reqtarget.RequestTarget, error) {
	return &authenticated{k.endpoint.WithCertificate(cert)}, nil
}

func (k *Keyserver) AuthenticateWithKerberosTickets() (reqtarget.RequestTarget, error) {
	return k.AuthenticateWithKerberosTicketsInCache("")
}

func (k *Keyserver) AuthenticateWithKerberosTicketsInCache(ticketcache string) (reqtarget.RequestTarget, error) {
	endpointURL, err := url.Parse(k.endpoint.BaseURL())
	if err != nil {
		return nil, err
	}

	return &knc.KncServer{Hostname: endpointURL.Hostname(), KerberosTicketCache: ticketcache}, nil
}

func (a *authenticated) SendRequests(reqs []reqtarget.Request) ([]string, error) {
	var outputs []string
	err := a.endpoint.PostJSON("/apirequest", reqs, &outputs)
	if err != nil {
		return nil, err
	}
	if len(outputs) != len(reqs) {
		return nil, errors.New("while finalizing response: wrong number of responses")
	}
	return outputs, nil
}
