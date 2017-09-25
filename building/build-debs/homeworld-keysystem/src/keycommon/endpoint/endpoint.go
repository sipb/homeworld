package endpoint

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type ServerEndpoint struct {
	rootCAs      *x509.CertPool
	baseURL      string
	extraHeaders map[string]string
	certificates []tls.Certificate
	timeout      time.Duration
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

func (s ServerEndpoint) BaseURL() string {
	return s.baseURL
}

func (s ServerEndpoint) WithHeader(key string, value string) ServerEndpoint {
	oldHeaders := s.extraHeaders
	s.extraHeaders = map[string]string{}
	for k, v := range oldHeaders {
		s.extraHeaders[k] = v
	}
	s.extraHeaders[key] = value
	return s
}

func (s ServerEndpoint) WithCertificate(cert tls.Certificate) ServerEndpoint {
	oldCerts := s.certificates
	s.certificates = make([]tls.Certificate, len(oldCerts))
	copy(s.certificates, oldCerts)
	s.certificates = append(s.certificates, cert)
	return s
}

func (s ServerEndpoint) Request(path string, method string, reqbody []byte) ([]byte, error) {
	if path[0] != '/' {
		return nil, errors.New("while validating request: path must be absolute")
	}
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      s.rootCAs,
				Certificates: s.certificates,
				MinVersion:   tls.VersionTLS12,
			},
			DisableCompression: true,
		},
		Timeout: s.timeout,
	}
	req, err := http.NewRequest(method, s.baseURL+path[1:], bytes.NewReader(reqbody))
	if err != nil {
		return nil, fmt.Errorf("while preparing request: %s", err.Error())
	}
	for k, v := range s.extraHeaders {
		req.Header.Set(k, v)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("while processing request: %s", err.Error())
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("while receiving response: %s", err.Error())
	}
	return body, nil
}

func (s ServerEndpoint) Get(path string) ([]byte, error) {
	return s.Request(path, "GET", nil)
}

func (s ServerEndpoint) PostJSON(path string, input interface{}, output interface{}) error {
	reqbody, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("while marshalling json for request: %s", err.Error())
	}
	body, err := s.Request(path, "POST", reqbody)
	if err != nil {
		return fmt.Errorf("while posting request: %s", err.Error())
	}
	err = json.Unmarshal(body, output)
	if err != nil {
		return fmt.Errorf("while unmarshalling json from response: %s", err.Error())
	}
	return nil
}
