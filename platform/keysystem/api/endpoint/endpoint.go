package endpoint

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
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
	client       *http.Client
}

func (s *ServerEndpoint) buildClient() {
	s.client = &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout: s.timeout,
			TLSClientConfig: &tls.Config{
				RootCAs:      s.rootCAs,
				Certificates: s.certificates,
				MinVersion:   tls.VersionTLS12,
			},
			TLSHandshakeTimeout: s.timeout,
			DisableCompression:  true,
		},
		Timeout: s.timeout,
	}
}

func NewServerEndpoint(url string, authorities *x509.CertPool) (ServerEndpoint, error) {
	if url == "" {
		return ServerEndpoint{}, errors.New("empty base URL")
	}
	if !strings.HasSuffix(url, "/") {
		return ServerEndpoint{}, errors.New("base URL must end in a slash")
	}
	ep := ServerEndpoint{rootCAs: authorities, baseURL: url, timeout: time.Second * 30}
	ep.buildClient()
	return ep, nil
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
	s.buildClient()
	return s
}

func (s ServerEndpoint) WithCertificate(cert tls.Certificate) ServerEndpoint {
	oldCerts := s.certificates
	s.certificates = make([]tls.Certificate, len(oldCerts))
	copy(s.certificates, oldCerts)
	s.certificates = append(s.certificates, cert)
	s.buildClient()
	return s
}

type OperationForbidden struct{}

func (o OperationForbidden) Error() string {
	return "operation forbidden by server"
}

func (s ServerEndpoint) Request(path string, method string, reqbody []byte) ([]byte, error) {
	if path[0] != '/' {
		return nil, errors.New("while validating request: path must be absolute")
	}
	req, err := http.NewRequest(method, s.baseURL+path[1:], bytes.NewReader(reqbody))
	if err != nil {
		return nil, errors.Wrap(err, "while preparing request")
	}
	for k, v := range s.extraHeaders {
		req.Header.Set(k, v)
	}
	response, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "while processing request")
	}
	if response.StatusCode != 200 {
		if response.StatusCode == 403 {
			return nil, OperationForbidden{}
		}
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "while receiving response")
	}
	return body, nil
}

func (s ServerEndpoint) Get(path string) ([]byte, error) {
	return s.Request(path, "GET", nil)
}

func (s ServerEndpoint) Post(path string, body []byte) ([]byte, error) {
	return s.Request(path, "POST", body)
}

func (s ServerEndpoint) PostJSON(path string, input interface{}, output interface{}) error {
	reqbody, err := json.Marshal(input)
	if err != nil {
		return errors.Wrap(err, "while marshalling json for request")
	}
	body, err := s.Request(path, "POST", reqbody)
	if err != nil {
		return errors.Wrap(err, "while posting request")
	}
	err = json.Unmarshal(body, output)
	if err != nil {
		return errors.Wrap(err, "while unmarshalling json from response")
	}
	return nil
}
