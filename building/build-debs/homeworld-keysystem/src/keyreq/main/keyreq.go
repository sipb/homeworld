package main

import (
	"log"
	"os"
	"keycommon/reqtarget"
	"io/ioutil"
	"keycommon/server"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
	"util/csrutil"
)

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
		logger.Fatal("keyreq should only be used by scripts that already know how to invoke it")
		return
	}
	switch os.Args[1] {
	case "check":
		if len(os.Args) < 4 {
			logger.Fatal("not enough parameters to keyreq ssh-cert <authority-path> <keyserver-domain>")
		}
		// just by calling this, we confirm that we do have access to the server. yay!
		_, _ = auth_kerberos(logger, os.Args[2], os.Args[3])
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
	case "kube-cert":
		if len(os.Args) < 7 {
			logger.Fatal("not enough parameters to keyreq kube-cert <authority-path> <keyserver-domain> <privkey-out> <cert-out> <ca-out>")
			return
		}
		ks, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
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
		err = ioutil.WriteFile(os.Args[4], privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("kubernetes")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[6], ca, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "etcd-cert":
		// TODO: deduplicate code
		if len(os.Args) < 7 {
			logger.Fatal("not enough parameters to keyreq etcd-cert <authority-path> <keyserver-domain> <privkey-out> <cert-out> <ca-out>")
			return
		}
		ks, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
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
		err = ioutil.WriteFile(os.Args[4], privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("etcd-server")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[6], ca, os.FileMode(0644))
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
		logger.Fatal("keyreq should only be used by scripts that already know how to invoke it")
	}
}
