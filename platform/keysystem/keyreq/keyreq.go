package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/util/csrutil"
)

const (
	ERR_UNKNOWN_FAILURE             = 1
	ERR_CANNOT_ESTABLISH_CONNECTION = 2
	ERR_NO_ACCESS                   = 3
	ERR_INVALID_CONFIG              = 254
	ERR_INVALID_INVOCATION          = 255
)

func get_keyserver(logger *log.Logger, authority_path string, keyserver_domain string) *server.Keyserver {
	authoritydata, err := ioutil.ReadFile(authority_path)
	if err != nil {
		logger.Printf("while loading authority: %s", err)
		os.Exit(ERR_INVALID_INVOCATION)
	}
	ks, err := server.NewKeyserver(authoritydata, keyserver_domain)
	if err != nil {
		logger.Print(err)
		os.Exit(ERR_INVALID_CONFIG)
	}
	return ks
}

func auth_kerberos(logger *log.Logger, authority_path string, keyserver_domain string) (*server.Keyserver, reqtarget.RequestTarget) {
	ks := get_keyserver(logger, authority_path, keyserver_domain)
	rt, err := ks.AuthenticateWithKerberosTickets()
	if err != nil {
		logger.Print(err)
		os.Exit(ERR_INVALID_INVOCATION)
	}
	// confirm that connection works
	_, err = rt.SendRequests([]reqtarget.Request{})
	if err != nil {
		logger.Print("failed to establish connection: ", err)
		os.Exit(ERR_CANNOT_ESTABLISH_CONNECTION)
	}
	return ks, rt
}

func main() {
	logger := log.New(os.Stderr, "[keyreq] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 2 {
		logger.Print("keyreq should only be used by scripts that already know how to invoke it")
		os.Exit(ERR_INVALID_INVOCATION)
	}

	switch os.Args[1] {
	case "check":
		if len(os.Args) < 4 {
			logger.Print("not enough parameters to keyreq check <authority-path> <keyserver-domain>")
			os.Exit(ERR_INVALID_INVOCATION)
		}
		// just by calling this, we confirm that we do have access to the server. yay!
		_, _ = auth_kerberos(logger, os.Args[2], os.Args[3])

	// TODO: deduplicate code
	case "ssh-cert":
		if len(os.Args) < 6 {
			logger.Print("not enough parameters to keyreq ssh-cert <authority-path> <keyserver-domain> <ssh-key-out> <ssh-cert-output>")
			os.Exit(ERR_INVALID_INVOCATION)
		}

		_, rt := auth_kerberos(logger, os.Args[2], os.Args[3])

		pkey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		pubkey_tmp, err := ssh.NewPublicKey(pkey.Public())
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		pubkey := ssh.MarshalAuthorizedKey(pubkey_tmp)

		req, err := reqtarget.SendRequest(rt, worldconfig.AccessSSHAPI, string(pubkey))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_NO_ACCESS)
		}
		if req == "" {
			logger.Print("empty result")
			os.Exit(ERR_UNKNOWN_FAILURE)
		}

		err = ioutil.WriteFile(os.Args[4], privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[4]+".pub", pubkey, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
	case "kube-cert":
		if len(os.Args) < 7 {
			logger.Print("not enough parameters to keyreq kube-cert <authority-path> <keyserver-domain> <privkey-out> <cert-out> <ca-out>")
			os.Exit(ERR_INVALID_INVOCATION)
		}
		ks, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		req, err := reqtarget.SendRequest(rt, worldconfig.AccessKubernetesAPI, string(csr))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_NO_ACCESS)
		}
		if req == "" {
			logger.Print("empty result")
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		err = ioutil.WriteFile(os.Args[4], privkey, os.FileMode(0600))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
		ca, err := ks.GetPubkey(worldconfig.KubernetesAuthority)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_CANNOT_ESTABLISH_CONNECTION)
		}
		err = ioutil.WriteFile(os.Args[6], ca, os.FileMode(0644))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
	case "etcd-cert":
		if len(os.Args) < 7 {
			logger.Print("not enough parameters to keyreq etcd-cert <authority-path> <keyserver-domain> <privkey-out> <cert-out> <ca-out>")
			os.Exit(ERR_INVALID_INVOCATION)
		}
		ks, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		req, err := reqtarget.SendRequest(rt, worldconfig.AccessEtcdAPI, string(csr))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_NO_ACCESS)
		}
		if req == "" {
			logger.Print("empty result")
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		err = ioutil.WriteFile(os.Args[4], privkey, os.FileMode(0600))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
		err = ioutil.WriteFile(os.Args[5], []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
		ca, err := ks.GetPubkey(worldconfig.EtcdServerAuthority)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_CANNOT_ESTABLISH_CONNECTION)
		}
		err = ioutil.WriteFile(os.Args[6], ca, os.FileMode(0644))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_INVALID_INVOCATION)
		}
	case "list-requests":
		if len(os.Args) != 4 {
			logger.Print("not enough parameters to keyreq list-requests <authority-path> <keyserver-domain>")
			os.Exit(ERR_INVALID_INVOCATION)
		}
		_, rt := auth_kerberos(logger, os.Args[2], os.Args[3])
		response, err := reqtarget.SendRequest(rt, worldconfig.ListAdmitRequestsAPI, "")
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_NO_ACCESS)
		}
		var requests map[string]admit.AdmitState
		err = json.Unmarshal([]byte(response), &requests)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		err = admit.PrintRequests(requests, os.Stdout)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
	case "admit":
		if len(os.Args) != 6 {
			logger.Print("not enough parameters to keyreq admit <authority-path> <keyserver-domain> <principal> <fingerprint>")
			os.Exit(ERR_INVALID_INVOCATION)
		}
		_, rt := auth_kerberos(logger, os.Args[2], os.Args[3])

		approval := admit.AdmitApproval{
			Principal:   os.Args[4],
			Fingerprint: os.Args[5],
		}
		data, err := json.Marshal(approval)
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
		response, err := reqtarget.SendRequest(rt, worldconfig.ApproveAdmitAPI, string(data))
		if err != nil {
			logger.Print(err)
			os.Exit(ERR_NO_ACCESS)
		}
		if response != "" {
			logger.Print("expected no response from approval API")
			os.Exit(ERR_UNKNOWN_FAILURE)
		}
	default:
		logger.Print("keyreq should only be used by scripts that already know how to invoke it")
		os.Exit(ERR_INVALID_INVOCATION)
	}
}
