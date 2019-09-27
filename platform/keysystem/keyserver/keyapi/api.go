package keyapi

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/operation"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/netutil"
)

type Keyserver interface {
	HandleAPIRequest(writer http.ResponseWriter, request *http.Request) error
	HandlePubRequest(writer http.ResponseWriter, authorityName string) error
	HandleStaticRequest(writer http.ResponseWriter, staticName string) error
	GetClientCAs() *x509.CertPool
	GetValidServerCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error)
}

type ConfiguredKeyserver struct {
	Context    *config.Context
	ServerKey  []byte
	ServerCert *tls.Certificate
	CertLock   sync.Mutex
	Logger     *log.Logger
}

func verifyAccountIP(account *account.Account, request *http.Request) error {
	ip, err := netutil.ParseRemoteAddressFromRequest(request)
	if err != nil {
		return err
	}
	allowedIp := account.LimitIP
	if allowedIp != nil && !allowedIp.Equal(ip) {
		return fmt.Errorf("attempt to interact with API from wrong IP address: %v instead of %v", ip, allowedIp)
	}
	return nil
}

func attemptAuthentication(context *config.Context, request *http.Request) (*account.Account, error) {
	verifiers := []verifier.Verifier{context.TokenVerifier, context.AuthenticationAuthority}

	for _, verif := range verifiers {
		if verif.HasAttempt(request) {
			principal, err := verif.Verify(request)
			if err != nil {
				return nil, err
			}
			ac, err := context.GetAccount(principal)
			if err != nil {
				return nil, err
			}
			if ac.DisableDirectAuth {
				return nil, fmt.Errorf("account has disabled direct authentication: %s", principal)
			}
			err = verifyAccountIP(ac, request)
			if err != nil {
				return nil, err
			}
			return ac, nil
		}
	}
	return nil, errors.New("no authentication method found in request")
}

func (k *ConfiguredKeyserver) GetClientCAs() *x509.CertPool {
	return k.Context.AuthenticationAuthority.ToCertPool()
}

const RenewalMargin = time.Minute * 5
const ValidityInterval = time.Hour * 24

func (k *ConfiguredKeyserver) GetValidServerCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	k.CertLock.Lock()
	defer k.CertLock.Unlock()

	if k.ServerCert != nil && time.Now().Add(RenewalMargin).Before(k.ServerCert.Leaf.NotAfter) {
		// we still have a valid certificate, so use that
		return k.ServerCert, nil
	}

	k.Logger.Printf("Signing new certificate for serving requests...")
	csr, err := csrutil.BuildTLSCSR(k.ServerKey)
	if err != nil {
		return nil, errors.Wrap(err, "while generating CSR")
	}
	cert, err := k.Context.ClusterCA.Sign(string(csr), true, ValidityInterval, "keyserver-autogen-tls", []string{k.Context.KeyserverDNS})
	if err != nil {
		return nil, errors.Wrap(err, "while signing CSR")
	}
	pair, err := tls.X509KeyPair([]byte(cert), k.ServerKey)
	if err != nil {
		return nil, errors.Wrap(err, "while reloading certificate")
	}
	pair.Leaf, err = x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "while pre-parsing certificate")
	}
	k.ServerCert = &pair
	k.Logger.Printf("New certificate will be valid until %v", k.ServerCert.Leaf.NotAfter)
	return k.ServerCert, nil
}

func (k *ConfiguredKeyserver) HandleAPIRequest(writer http.ResponseWriter, request *http.Request) error {
	requestBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return err
	}
	ac, err := attemptAuthentication(k.Context, request)
	if err != nil {
		return err
	}
	response, err := operation.InvokeAPIOperationSet(ac, k.Context, requestBody, k.Logger)
	if err != nil {
		return err
	}
	_, err = writer.Write(response)
	return err
}

func (k *ConfiguredKeyserver) HandlePubRequest(writer http.ResponseWriter, authorityName string) error {
	authority := k.Context.Authorities[authorityName]
	if authority == nil {
		return fmt.Errorf("no such authority %s", authorityName)
	}
	_, err := writer.Write(authority.GetPublicKey())
	return err
}

func (k *ConfiguredKeyserver) HandleStaticRequest(writer http.ResponseWriter, staticName string) error {
	file, found := k.Context.StaticFiles[staticName]
	if !found || file.Filepath == "" {
		return fmt.Errorf("no such static file %s", staticName)
	}
	contents, err := ioutil.ReadFile(file.Filepath)
	if err != nil {
		return err // odd; we didn't see this earlier
	}
	_, err = writer.Write(contents)
	return err
}
