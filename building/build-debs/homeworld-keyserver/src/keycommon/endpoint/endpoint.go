package endpoint

import (
	"crypto/x509"
	"errors"
	"strings"
	"fmt"
	"io/ioutil"
	"net/http"
	"crypto/tls"
	"bytes"
	"encoding/json"
	"time"
)

type ServerEndpoint struct {
	rootCAs *x509.CertPool
	baseURL string
	extraHeaders map[string]string
	certificates []tls.Certificate
	timeout time.Duration
}

func NewServerEndpoint(url string, authorities *x509.CertPool) (ServerEndpoint, error) {
	if url == "" {
		return ServerEndpoint{}, errors.New("empty base URL")
	}
	if !strings.HasSuffix(url, "/") {
		return ServerEndpoint{}, errors.New("base URL must end in a slash")
	}
	return ServerEndpoint{rootCAs: authorities, baseURL: url, timeout: time.Second * 30}, nil
}

func (s ServerEndpoint) WithHeader(key string, value string) ServerEndpoint {
	out := s
	out.extraHeaders = map[string]string {}
	for k, v := range s.extraHeaders {
		out.extraHeaders[k] = v
	}
	out.extraHeaders[key] = value
	return out
}

func (s ServerEndpoint) WithCertificate(cert tls.Certificate) ServerEndpoint {
	out := s
	out.certificates = make([]tls.Certificate, len(s.certificates))
	copy(out.certificates, s.certificates)
	out.certificates = append(out.certificates, cert)
	return out
}

func (s ServerEndpoint) Request(path string, method string, reqbody []byte) ([]byte, error) {
	if path[0] != '/' {
		return nil, errors.New("Path must be absolute")
	}
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: s.rootCAs,
				Certificates: s.certificates,
			},
		},
		Timeout: s.timeout,
	}
	req, err := http.NewRequest(method, s.baseURL + path[1:], bytes.NewReader(reqbody))
	if err != nil {
		return nil, err
	}
	for k, v := range s.extraHeaders {
		req.Header.Set(k, v)
	}
	response, err := client.Do(req)
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

func (s ServerEndpoint) Get(path string) ([]byte, error) {
	return s.Request(path, "GET", nil)
}

func (s ServerEndpoint) PostJSON(path string, input interface{}, output interface{}) error {
	reqbody, err := json.Marshal(input)
	if err != nil {
		return err
	}
	body, err := s.Request(path, "POST", reqbody)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, output)
	if err != nil {
		return err
	}
	return nil
}
