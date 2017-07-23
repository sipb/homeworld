package main

import (
	"fmt"
	"net/http"
	"log"
	"io/ioutil"
	"time"
	"crypto/tls"
	"crypto/x509"
	"strings"
)

const (
	cluster_conf       = "/etc/hyades/admission/config/cluster.conf"
	node_base          = "/etc/hyades/admission/config/node-%s.conf"
	keyfile            = "/etc/hyades/admission/admission.key"
	certfile           = "/etc/hyades/admission/admission.pem"
	userca             = "/etc/hyades/admission/bootstrap_client_ca.pem"
	signingca          = "/etc/hyades/admission/ssh_host_ca"
	authca             = "/etc/hyades/admission/ssh_user_ca.pub"
	validityInterval   = time.Hour * 24 * 7 * 4 // generated certificates are valid for four weeks
)

func isValidName(name string, domain bool) bool {
	for _, r := range name {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || (domain && r == '.')
		if !valid {
			return false
		}
	}
	if domain && !strings.Contains(name, ".") {
		return false
	}
	return len(name) >= 4
}

func getKeyFromConfig(config []byte, key string) (string, error) {
	for _, line := range strings.Split(string(config), "\n") {
		if strings.HasPrefix(line, key + "=") {
			return line[len(key) + 1:], nil
		}
	}
	return "", fmt.Errorf("Cannot find key %v in configuration file", key)
}

func ServeWithTLS(addr string, client_ca_file string, server_cert string, server_key string) error {
	server := &http.Server{
		Addr: addr,
		Handler: nil,
		TLSConfig: &tls.Config{
			ClientAuth: tls.VerifyClientCertIfGiven,
			ClientCAs:  x509.NewCertPool(),
		},
	}

	authority, err := ioutil.ReadFile(client_ca_file)
	if err != nil {
		return err
	}
	if !server.TLSConfig.ClientCAs.AppendCertsFromPEM(authority) {
		return fmt.Errorf("Could not process client CA")
	}

	return server.ListenAndServeTLS(server_cert, server_key)
}

func main() {
	handler := NewTokenHandler()

	ca_user_pubkey, err := ioutil.ReadFile(authca)

	if err != nil {
		log.Fatal(err)
	}

	cluster_config, err := ioutil.ReadFile(cluster_conf)
	if err != nil {
		log.Fatal(err)
	}

	cluster_domain, err := getKeyFromConfig(cluster_config, "DOMAIN")
	if err != nil {
		log.Fatal(err)
	}

	if !isValidName(cluster_domain, true) {
		log.Fatalf("Invalid domain: '%v'", cluster_domain)
	}

	log.Printf("Cluster domain: %v\n", cluster_domain)

	ssh_ca, err := LoadSSHCertificateAuthority(signingca, validityInterval)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/bootstrap/", func(writer http.ResponseWriter, request *http.Request) {
		if len(request.TLS.VerifiedChains) == 0 || len(request.TLS.VerifiedChains[0]) == 0 {
			http.Error(writer, "Valid certificate required for generating bootstrap tokens", http.StatusForbidden)
			return
		}
		firstCert := request.TLS.VerifiedChains[0][0]
		hostname := request.URL.Path[len("/bootstrap/"):]
		if !isValidName(hostname, false) {
			http.Error(writer, "Invalid hostname requested", http.StatusNotFound)
			return
		}
		configuration, err := ioutil.ReadFile(fmt.Sprintf(node_base, hostname))
		if err != nil {
			http.Error(writer, "No configuration available for hostname", http.StatusNotFound)
			return
		}
		token := handler.GrantToken(hostname, configuration, firstCert.Subject.CommonName)
		log.Printf("Granted bootstrap token to %v for hostname %v\n", firstCert.Subject.CommonName, hostname)
		fmt.Fprintf(writer, "Token: %v\n", token)
	})
	http.HandleFunc("/config/ssh_user_ca.pub", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(ca_user_pubkey)
	})
	http.HandleFunc("/config/cluster.conf", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(cluster_config)
	})
	http.HandleFunc("/config/local.conf", handler.wrapHandler(
		func(tdata tokenData, writer http.ResponseWriter, request *http.Request) bool {
			writer.Write(tdata.configuration)
			return false // never claim
		}))
	http.HandleFunc("/certificates", handler.wrapHandler(
		func(tdata tokenData, writer http.ResponseWriter, request *http.Request) bool {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				http.Error(writer, fmt.Sprintf("Could not read request body: %v", err), http.StatusInternalServerError)
				return false
			}

			keyid := fmt.Sprintf("admitted-by:%s@%d", tdata.admin, uint64(time.Now().Unix()))
			hostnames := []string{tdata.hostname, tdata.hostname + "." + cluster_domain}

			output, err := ssh_ca.SignEncodedHostPubkeys(data, keyid, hostnames)
			if err != nil {
				http.Error(writer, fmt.Sprintf("Could not sign pubkeys: %v", err), http.StatusInternalServerError)
				return false
			}
			writer.Write(output)
			return true
		}))

	log.Fatal(ServeWithTLS(":2557", userca, certfile, keyfile))
}
