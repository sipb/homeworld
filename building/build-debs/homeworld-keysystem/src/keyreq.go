package main

import (
	"log"
	"os"
	"keycommon"
	"os/user"
	"path"
	"keycommon/reqtarget"
	"io/ioutil"
	"keycommon/server"
	"strings"
	"golang.org/x/crypto/ssh"
	"encoding/base64"
	"fmt"
	"errors"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
	"util/csrutil"
)

func usage(logger *log.Logger) {
	logger.Fatal("invalid usage\nusage: keyreq ssh\n  request a SSH cert for ~/.ssh/id_rsa.pub, placing it in ~/.ssh/id_rsa-cert.pub\n" +
		"usage: keyreq ssh-host\n  place the @ca-certificates directive into ~/.ssh/known_hosts\n" +
		"usage: keyreq kube\n  generate a key and request a kubernetes auth cert, placing them in ~/.homeworld/kube.{pem,key}\n" +
		"  (also, put the kubernetes CA cert in ~/.homeworld/kube-ca.pem)\n" +
		"usage: keyreq etcd\n  generate a key and request an etcd auth cert, placing them in ~/.homeworld/etcd.{pem,key}\n" +
		"  (also, put the etcd CA cert in ~/.homeworld/etcd-ca.pem)\n" +
		"usage: keyreq bootstrap <server-principal>\n  request a bootstrap token for a server\n\n" +
	    "these depend on ~/.homeworld/keyreq.yaml being configured properly\n")
}

func homedir_DEPRECATED(logger *log.Logger) string {
	usr, err := user.Current()
	if err != nil {
		logger.Fatal(err)
	}
	return usr.HomeDir
}

func get_keyserver(logger *log.Logger, authority_path string, keyserver_domain string) *server.Keyserver {
	authoritydata, err := ioutil.ReadFile(authority_path)
	if err != nil {
		logger.Fatalf("while loading authority: %s", err)
	}
	ks, err := server.NewKeyserver(authoritydata, keyserver_domain)
	if err != nil {
		logger.Fatal(err)
	}
	return ks
}

func auth_kerberos(logger *log.Logger, authority_path string, keyserver_domain string) (*server.Keyserver, reqtarget.RequestTarget) {
	ks := get_keyserver(logger, authority_path, keyserver_domain)
	rt, err := ks.AuthenticateWithKerberosTickets()
	if err != nil {
		logger.Fatal(err)
	}
	// confirm that connection works
	_, err = rt.SendRequests([]reqtarget.Request{})
	if err != nil {
		logger.Fatal("failed to check connection: ", err)
	}
	return ks, rt
}

func main() {
	logger := log.New(os.Stderr, "[keyreq] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 2 {
		usage(logger)
		return
	}
	switch os.Args[1] {
	case "ssh-cert": // called programmatically
		if len(os.Args) < 6 {
			logger.Fatal("not enough parameters to keyreq ssh-cert <authority-path> <keyserver-domain> <ssh.pub-in> <ssh-cert-output>")
		}
		_, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
		ssh_pubkey, err := ioutil.ReadFile(os.Args[4])
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-ssh", string(ssh_pubkey))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "kube":
		ks, rt := auth_kerberos(logger)
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Fatal(err)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-kubernetes", string(csr))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "kube.key"), privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "kube.pem"), []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("kubernetes")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "kube-ca.pem"), ca, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "etcd":
		// TODO: deduplicate code
		ks, rt := auth_kerberos(logger)
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Fatal(err)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-etcd", string(csr))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "etcd.key"), privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "etcd.pem"), []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("etcd-server")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir_DEPRECATED(logger), ".homeworld", "etcd-ca.pem"), ca, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "bootstrap-token":
		if len(os.Args) < 5 {
			logger.Fatal("not enough parameters to keyreq bootstrap-token <authority-path> <keyserver-domain> <principal>")
			return
		}
		_, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
		token, err := reqtarget.SendRequest(rt, "bootstrap", os.Args[4])
		if err != nil {
			logger.Fatal(err)
		}
		os.Stdout.WriteString(token + "\n")
	default:
		usage(logger)
	}
}
