package keyserver

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"authorities"
	"config"
	"account"
	"token/auth"
	"util"
	"net"
)

func verifyAccountIP(account *account.Account, request *http.Request) error {
	ip, err := util.ParseRemoteAddressFromRequest(request)
	if err != nil {
		return err
	}
	allowed_ip_str, found := account.Metadata["ip"]
	if !found {
		return fmt.Errorf("No allowed IP for bootstrap target.")
	}
	allowed_ip := net.ParseIP(allowed_ip_str)
	if allowed_ip == nil {
		return fmt.Errorf("IP address malformed for bootstrap target.")
	}
	if !allowed_ip.Equal(ip) {
		return fmt.Errorf("Attempt to use bootstrap token from wrong IP address.")
	}
	return nil
}

func attemptAuthentication(context *config.Context, request *http.Request) (*account.Account, error) {
	if auth.HasTokenAuthHeader(request) {
		// Auth with a token.
		principal, err := auth.Authenticate(context.TokenRegistry, request)
		if err != nil {
			return nil, err
		}
		ac, err := context.GetAccount(principal)
		if err != nil {
			return nil, err
		}
		err = verifyAccountIP(ac, request)
		if err != nil {
			return nil, err
		}
		return ac, nil
	} else {
		// Auth with a cert.
		principal, err := context.Authenticator.Verify(request) // Try the main authentication authority
		if err != nil {
			return nil, err
		}
		ac, err := context.GetAccount(principal)
		if err != nil {
			return nil, err
		}
		authority, ok := ac.GrantingAuthority.(*authorities.TLSAuthority)
		if !ok || !authority.Equal(context.Authenticator) {
			return nil, fmt.Errorf("Mismatched authority during authentication")
		}
		return ac, nil
	}
}

func handleAPIRequest(context *config.Context, writer http.ResponseWriter, request *http.Request) error {
	requestBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return err
	}
	ac, err := attemptAuthentication(context, request)
	if err != nil {
		return err
	}
	response, err := ac.InvokeAPIOperationSet(requestBody)
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

func Run(configfile string) error {
	context, err := config.LoadConfig(configfile)
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

	server := &http.Server{
		Addr: ":20557",
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth: tls.VerifyClientCertIfGiven,
			ClientCAs:  x509.NewCertPool(),
			Certificates: []tls.Certificate { context.ServerTLS.ToHTTPSCert() },
		},
	}

	context.Authenticator.Register(server.TLSConfig.ClientCAs)

	return server.ListenAndServeTLS("", "")
}
