package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sipb/homeworld/platform/util/certutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

type Config struct {
	Address          string
	ApiserverAddress string
	EmailAuthDomain  string
	UpstreamCA       *x509.Certificate
	KubeCA           []byte
	IssuerCert       *x509.Certificate
	IssuerKey        *rsa.PrivateKey
	ServerTLS        tls.Certificate
}

const CertificateLifespan = time.Hour * 24 * 90

// TODO: generate this as actual YAML, not through text substitution
const pageTemplate = `# place this file in ~/.kube/config
# keep this file secret; it allows authenticating to the Hyades cluster as you
# this certificate will last 90 days, at which point you will need to request a new one

current-context: hyades
apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority-data: "{{.AuthorityBase64}}"
    server: https://{{.Server}}:443
  name: hyades-cluster
users:
- name: kubectl-auth
  user:
    client-certificate-data: "{{.CertBase64}}"
    client-key-data: "{{.KeyBase64}}"
contexts:
- context:
    cluster: hyades-cluster
    namespace: "user-{{.User}}"
    user: kubectl-auth
  name: hyades
`

const characterWhitelist = "^[-a-zA-Z0-9_/~+=.]+$"

func validateCharset(s string) error {
	match, err := regexp.MatchString(characterWhitelist, s)
	if err != nil {
		return err
	}
	if !match {
		return fmt.Errorf("not a valid string: %v", []byte(s))
	}
	return nil
}

type response struct {
	User            string
	KeyBase64       string
	CertBase64      string
	AuthorityBase64 string
	Server          string
}

func (r *response) Validate() error {
	for _, s := range []string{r.User, r.KeyBase64, r.CertBase64, r.AuthorityBase64, r.Server} {
		if err := validateCharset(s); err != nil {
			return err
		}
	}
	return nil
}

var oidEmailAddress = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}

func isOIDEqual(a asn1.ObjectIdentifier, b asn1.ObjectIdentifier) bool {
	if len(a) != len(b) {
		return false
	}
	for i, av := range a {
		if av != b[i] {
			return false
		}
	}
	return true
}

func (c *Config) ExtractEmail(name pkix.Name) (string, error) {
	for _, attr := range name.Names {
		if isOIDEqual(attr.Type, oidEmailAddress) {
			result, ok := attr.Value.(string)
			if !ok {
				return "", errors.New("value in AttributeTypeAndValue was not a string")
			}
			if !strings.HasSuffix(result, "@"+c.EmailAuthDomain) {
				return "", fmt.Errorf("email '%s' had no suffix '%s'", result, "@"+c.EmailAuthDomain)
			}
			return result[:len(result)-len("@"+c.EmailAuthDomain)], nil
		}
	}
	return "", errors.New("cannot find email attribute in certificate name")
}

func (c *Config) Authenticate(request *http.Request) (string, error) {
	if request.TLS == nil {
		return "", errors.New("no TLS")
	}
	if len(request.TLS.VerifiedChains) == 0 {
		return "", errors.New("no verified client chains")
	}
	if len(request.TLS.VerifiedChains[0]) == 0 {
		return "", errors.New("no certificates in verified chain")
	}
	firstCert := request.TLS.VerifiedChains[0][0]
	chains, err := firstCert.Verify(x509.VerifyOptions{
		Roots:     c.GetClientCAs(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if len(chains) == 0 || err != nil {
		return "", fmt.Errorf("certificate not valid under this authority: %v", err)
	}
	return c.ExtractEmail(firstCert.Subject)
}

func (c *Config) CertGen(key crypto.PublicKey, user string) ([]byte, error) {
	issueAt := time.Now()
	certTemplate := &x509.Certificate{
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  false,
		MaxPathLen:            0,
		MaxPathLenZero:        true,

		NotBefore: issueAt,
		NotAfter:  issueAt.Add(CertificateLifespan),

		Subject: pkix.Name{CommonName: "user:" + user},
	}
	return certutil.FinishCertificate(certTemplate, c.IssuerCert, key, c.IssuerKey)
}

func (c *Config) HandleRequest(writer http.ResponseWriter, request *http.Request) error {
	user, err := c.Authenticate(request)
	if err != nil {
		return err
	}
	templ, err := template.New("kubeconfig").Parse(pageTemplate)
	if err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	keyx := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := c.CertGen(key.Public(), user)
	resp := response{
		User:            user,
		Server:          c.ApiserverAddress,
		AuthorityBase64: base64.StdEncoding.EncodeToString(c.KubeCA),
		KeyBase64:       base64.StdEncoding.EncodeToString(keyx),
		CertBase64:      base64.StdEncoding.EncodeToString(cert),
	}
	if err := resp.Validate(); err != nil {
		return err
	}
	return templ.Execute(writer, resp)
}

func (c *Config) GetClientCAs() *x509.CertPool {
	cp := x509.NewCertPool()
	cp.AddCert(c.UpstreamCA)
	return cp
}

func Launch(c *Config) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		err := c.HandleRequest(writer, request)
		if err != nil {
			log.Printf("Grant request failed with error: %s", err)
			http.Error(writer, "Grant processing failed. See server logs for details.", http.StatusBadRequest)
		}
	})

	server := &http.Server{
		Addr:    c.Address,
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			ClientCAs:    c.GetClientCAs(),
			Certificates: []tls.Certificate{c.ServerTLS},
			MinVersion:   tls.VersionTLS12,
			NextProtos:   []string{"http/1.1", "h2"},
		},
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(ln, server.TLSConfig)
	return server.Serve(tlsListener)
}

func LoadConfig(rawArgs []string) (*Config, error) {
	args := map[string]string{
		"upstream-ca":  "",
		"kube-ca":      "",
		"server-key":   "",
		"server-cert":  "",
		"issuer-key":   "",
		"issuer-cert":  "",
		"apiserver":    "",
		"address":      ":443",
		"email-domain": "",
	}
	for _, arg := range rawArgs {
		pts := strings.SplitN(arg, "=", 2)
		if len(pts) < 2 {
			log.Fatalln("expected argument to be key/value format")
		}
		if _, ok := args[pts[0]]; !ok {
			log.Fatalf("unexpected argument: %s\n", pts[0])
		}
		args[pts[0]] = pts[1]
	}

	c := &Config{
		Address:          args["address"],
		ApiserverAddress: args["apiserver"],
		EmailAuthDomain:  args["email-domain"],
	}

	var err error

	if c.Address == "" {
		return nil, errors.New("no bind address specified")
	}

	if c.ApiserverAddress == "" {
		return nil, errors.New("no apiserver address specified")
	}

	if c.EmailAuthDomain == "" {
		return nil, errors.New("no email authentication domain specified")
	}

	c.UpstreamCA, err = wraputil.LoadX509FromPath(args["upstream-ca"])
	if err != nil {
		return nil, err
	}

	c.KubeCA, err = ioutil.ReadFile(args["kube-ca"])
	if err != nil {
		return nil, err
	}

	c.IssuerCert, err = wraputil.LoadX509FromPath(args["issuer-cert"])
	if err != nil {
		return nil, err
	}

	c.IssuerKey, err = wraputil.LoadRSAKeyFromPath(args["issuer-key"])
	if err != nil {
		return nil, err
	}

	c.ServerTLS, err = tls.LoadX509KeyPair(args["server-cert"], args["server-key"])
	if err != nil {
		return nil, err
	}

	return c, nil
}

func main() {
	conf, err := LoadConfig(os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalln(Launch(conf))
}
