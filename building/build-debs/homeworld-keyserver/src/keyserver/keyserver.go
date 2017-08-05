package keyserver

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"authorities"
	"token"
	"config"
	"account"
	"errors"
)

func attemptAuthentication(context *config.Context, request *http.Request) (account.Account, error) {
	var authorityUsed authorities.Authority
	// First, try with a token.
	principal, err := context.TokenRegistry.Verify(request)
	if token.IsNoTokenError(err) { // Nope -- they didn't try to authenticate with a token.
		principal, err = context.Authenticator.Verify(request) // Try the main authentication authority
		if err != nil {
			return nil, err
		}
		authorityUsed = context.Authenticator
		if authorityUsed == nil {
			return nil, fmt.Errorf("Missing authenticator in internal struct.")
		}
	} else if err != nil { // some error besides lacking a token, which should actually cause this to fail.
		return nil, err
	}
	account, found := context.Accounts[principal]
	if !found {
		return nil, fmt.Errorf("No such principal in database: %s", account)
	}
	if authorityUsed != nil && account.GrantingAuthority != authorityUsed {
		return nil, fmt.Errorf("Mismatched authority during authentication")
	}
	if account.Principal != principal {
		return nil, fmt.Errorf("Mismatched principal during authentication")
	}
	return account, nil
}

func handleAPIRequest(context *config.Context, writer http.ResponseWriter, request *http.Request) error {
	requestBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return err
	}
	account, err := attemptAuthentication(context, request)
	if err != nil {
		return err
	}
	response, err := account.InvokeAPIOperationSet(context, requestBody)
	if err != nil {
		return err
	}
	_, err = writer.Write(response)
	return err
}

func handlePubRequest(context *config.Context, writer http.ResponseWriter, request *http.Request) error {
	authorityName := request.URL.Path[len("/pub/"):]
	authority := context.Authorities[authorityName]
	if authority == nil {
		return fmt.Errorf("No such authority %s", authorityName)
	}
	_, err := writer.Write(authority.GetPublicKey())
	return err
}

func handleStaticRequest(context *config.Context, writer http.ResponseWriter, request *http.Request) error {
	staticName := request.URL.Path[len("/pub/"):]
	file, found := context.StaticFiles[staticName]
	if !found || file.Filepath == "" {
		return fmt.Errorf("No such static file %s", staticName)
	}
	contents, err := ioutil.ReadFile(file.Filepath)
	if err != nil {
		return err // odd; we didn't see this earlier
	}
	_, err = writer.Write(contents)
	return err
}

func run() error {
	context, err := config.LoadDefaultConfig()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/apirequest", func(writer http.ResponseWriter, request *http.Request) {
		err := handleAPIRequest(context, writer, request)
		if err != nil {
			log.Println("API request failed with error: %s", err)
			http.Error(writer, "Request processing failed. See server logs for details.", http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/pub/", func(writer http.ResponseWriter, request *http.Request) {
		err := handlePubRequest(context, writer, request)
		if err != nil {
			log.Println("Public request failed with error: %s", err)
			http.Error(writer, "Request processing failed: " + err.Error(), http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/static/", func(writer http.ResponseWriter, request *http.Request) {
		err := handleStaticRequest(context, writer, request)
		if err != nil {
			log.Println("Static request failed with error: %s", err)
			http.Error(writer, "Request processing failed: " + err.Error(), http.StatusBadRequest)
		}
	})

	authenticator := context.Authenticator.(*authorities.TLSAuthority)
	if authenticator == nil {
		return errors.New("Authenticator is not a TLS authority!")
	}
	servertls := context.ServerTLS.(*authorities.TLSAuthority)
	if authenticator == nil {
		return errors.New("ServerTLS is not a TLS authority!")
	}

	server := &http.Server{
		Addr: ":20557",
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth: tls.VerifyClientCertIfGiven,
			ClientCAs:  x509.NewCertPool(),
			Certificates: []tls.Certificate { servertls.ToHTTPSCert() },
		},
	}

	authenticator.Register(server.TLSConfig.ClientCAs)

	return server.ListenAndServeTLS("", "")
}

func main() {
	log.Fatal(run())
}
