package keyserver

import (
	"account"
	"config"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"operation"
	"util"
	"verifier"
)

func verifyAccountIP(account *account.Account, request *http.Request) error {
	ip, err := util.ParseRemoteAddressFromRequest(request)
	if err != nil {
		return err
	}
	allowed_ip := account.LimitIP
	if allowed_ip != nil && !allowed_ip.Equal(ip) {
		return fmt.Errorf("Attempt to use bootstrap token from wrong IP address.")
	}
	return nil
}

func attemptAuthentication(context *config.Context, request *http.Request) (*account.Account, error) {
	verifiers := []verifier.Verifier{context.TokenVerifier, context.AuthenticationAuthority.AsVerifier()}

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
				return nil, fmt.Errorf("Account has disabled direct authentication: %s", principal)
			}
			err = verifyAccountIP(ac, request)
			if err != nil {
				return nil, err
			}
			return ac, nil
		}
	}
	return nil, fmt.Errorf("No authentication method found in request.")
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
	response, err := operation.InvokeAPIOperationSet(ac, context, requestBody)
	if err != nil {
		return err
	}
	_, err = writer.Write(response)
	return err
}

func handlePubRequest(context *config.Context, writer http.ResponseWriter, authorityName string) error {
	authority := context.Authorities[authorityName]
	if authority == nil {
		return fmt.Errorf("No such authority %s", authorityName)
	}
	_, err := writer.Write(authority.GetPublicKey())
	return err
}

func handleStaticRequest(context *config.Context, writer http.ResponseWriter, staticName string) error {
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
		err := handlePubRequest(context, writer, request.URL.Path[len("/pub/"):])
		if err != nil {
			log.Println("Public request failed with error: %s", err)
			http.Error(writer, "Request processing failed: "+err.Error(), http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/static/", func(writer http.ResponseWriter, request *http.Request) {
		err := handleStaticRequest(context, writer, request.URL.Path[len("/static/"):])
		if err != nil {
			log.Println("Static request failed with error: %s", err)
			http.Error(writer, "Request processing failed: "+err.Error(), http.StatusBadRequest)
		}
	})

	server := &http.Server{
		Addr:    ":20557",
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			ClientCAs:    context.AuthenticationAuthority.ToCertPool(),
			Certificates: []tls.Certificate{context.ServerTLS.ToHTTPSCert()},
		},
	}

	return server.ListenAndServeTLS("", "")
}
