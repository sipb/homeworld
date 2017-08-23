package keycommon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"bytes"
	"io/ioutil"
	"golang.org/x/crypto/ssh"
)

type Keyserver struct {
	client  http.Client
	baseurl string
}

func parseTLSCertificate(data []byte) (*x509.Certificate, error) {
	block, rest := pem.Decode(data)
	if len(rest) != 0 {
		return nil, fmt.Errorf("Unexpected data found when looking for PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("Expected PEM block for CERTIFICATE, not %s", block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func NewKeyserver(authority []byte, hostname string) (*Keyserver, error) {
	cert, err := parseTLSCertificate(authority)
	if err != nil {
		return nil, fmt.Errorf("While parsing authority certificate: %s", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)

	ks := &Keyserver{}
	ks.baseurl = "https://" + hostname + ":20557"
	ks.client = http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}}

	return ks, nil
}

func (k *Keyserver) GetStatic(staticname string) ([]byte, error) {
	if staticname == "" {
		return nil, fmt.Errorf("Static filename is empty.")
	}
	return k.fetchPath("/static/" + staticname)
}

func (k *Keyserver) GetPubkey(authorityname string) ([]byte, error) {
	if authorityname == "" {
		return nil, fmt.Errorf("Authority name is empty.")
	}
	return k.fetchPath("/pub/" + authorityname)
}

func (k *Keyserver) GetPubkeyAsTLS(authorityname string) (*x509.Certificate, error) {
	data, err := k.GetPubkey(authorityname)
	if err != nil {
		return nil, err
	}
	cert, err := parseTLSCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("While decoding TLS pubkey: %s", err)
	}
	return cert, nil
}

func (k *Keyserver) GetPubkeyAsSSH(authorityname string) (ssh.PublicKey, error) {
	data, err := k.GetPubkey(authorityname)
	if err != nil {
		return nil, err
	}
	cert, _, _, rest, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, fmt.Errorf("While decoding SSH pubkey: %s", err)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("Found unexpected trailing data while decoding SSH pubkey")
	}
	return cert, nil
}

func (k *Keyserver) fetchPath(path string) ([]byte, error) {
	if path[0] != '/' {
		return nil, errors.New("Path must be absolute")
	}
	response, err := k.client.Get(k.baseurl + path)
	if err != nil {
		return nil, fmt.Errorf("While processing request: %s", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("While receiving response: %s", err)
	}
	return body, nil
}

type tokenAuth struct {
	k *Keyserver
	token string
}

func (k *Keyserver) AuthenticateWithToken(token string) (RequestTarget, error) {
	if token == "" {
		return nil, errors.New("Invalid token.")
	}
	return &tokenAuth{ k, token }, nil
}

func (a *tokenAuth) auth(request *http.Request) error {
	request.Header.Set("X-Bootstrap-Token", a.token)
	return nil
}

func (a *tokenAuth) SendRequests(reqs []Request) ([]string, error) {
	return a.k.sendRequests(reqs, a.k.client, a.auth)
}

type certAuth struct {
	k *Keyserver
	client http.Client
	cert tls.Certificate
}

func (k *Keyserver) AuthenticateWithCert(cert tls.Certificate) (RequestTarget, error) {
	client := k.client
	txport := *client.Transport.(*http.Transport)
	cconf := *txport.TLSClientConfig
	cconf.Certificates = append(cconf.Certificates, cert)
	txport.TLSClientConfig = &cconf
	client.Transport = &txport

	return &certAuth{ k,client, cert }, nil
}

func (a *certAuth) SendRequests(reqs []Request) ([]string, error) {
	return a.k.sendRequests(reqs, a.client, func(_ *http.Request) error { return nil })
}

func SendRequest(a RequestTarget, api string, body string) (string, error) {
	strs, err := a.SendRequests([]Request{{API: api, Body: body}})
	if err != nil {
		return "", err
	}
	if len(strs) != 1 {
		return "", fmt.Errorf("Wrong number of results: %d != 1", len(strs))
	}
	return strs[0], nil
}

func (k *Keyserver) sendRequests(reqs []Request, client http.Client, authmethod func(*http.Request) error) ([]string, error) {
	jsonready := make([]struct{ api string; body string }, len(reqs))
	for i, req := range reqs {
		jsonready[i].api = req.API
		jsonready[i].body = req.Body
	}
	reqdata, err := json.Marshal(jsonready)
	if err != nil {
		return nil, fmt.Errorf("While encoding request: %s", err)
	}
	request, err := http.NewRequest("POST", k.baseurl + "/apirequest", bytes.NewReader(reqdata))
	if err != nil {
		return nil, fmt.Errorf("While preparing request: %s", err)
	}
	err = authmethod(request)
	if err != nil {
		return nil, fmt.Errorf("While preparing authentication: %s", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("While processing request: %s", err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Unexpected status code: %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("While receiving response: %s", err)
	}
	outputs := []string {}
	err = json.Unmarshal(body, &outputs)
	if err != nil {
		return nil ,fmt.Errorf("While decoding response: %s", err)
	}
	if len(outputs) != len(reqs) {
		return nil, fmt.Errorf("While finalizing response: wrong number of responses")
	}
	return outputs, nil
}
