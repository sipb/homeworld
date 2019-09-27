package keyapi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/operation"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
	"github.com/sipb/homeworld/platform/util/netutil"
)

type Keyserver interface {
	HandleAPIRequest(writer http.ResponseWriter, request *http.Request) error
	HandlePubRequest(writer http.ResponseWriter, authorityName string) error
	HandleStaticRequest(writer http.ResponseWriter, staticName string) error
	GetClientCAs() *x509.CertPool
	GetServerCert() tls.Certificate
}

type ConfiguredKeyserver struct {
	Context    *config.Context
	ServerCert tls.Certificate
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

func (k *ConfiguredKeyserver) GetServerCert() tls.Certificate {
	return k.ServerCert
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
