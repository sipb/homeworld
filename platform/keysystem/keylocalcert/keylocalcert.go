package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func main() {
	logger := log.New(os.Stderr, "[keylocalcert] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 8 {
		logger.Fatal("usage: keylocalcert <ca-key> <ca-cert> <principal> <lifespan> <out-key> <out-cert> <organizations>\n  generates a kubernetes or etcd certificate directly")
	}
	keydata, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	certdata, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		logger.Fatal(err)
	}
	isSSH := !wraputil.IsPEMBlock(certdata)
	var authority authorities.Authority
	if isSSH {
		authority, err = authorities.LoadSSHAuthority(keydata, certdata)
	} else {
		authority, err = authorities.LoadTLSAuthority(keydata, certdata)
	}
	if err != nil {
		logger.Fatal(err)
	}
	commonname := os.Args[3]
	lifespan, err := time.ParseDuration(os.Args[4])
	if err != nil {
		logger.Fatal(err)
	}
	var organizations []string
	if len(os.Args[7]) > 0 {
		organizations = strings.Split(os.Args[7], ",")
	}

	pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
	if err != nil {
		logger.Fatal(err)
	}
	privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})

	var csr []byte
	if isSSH {
		pubkey, err := ssh.NewPublicKey(pkey.Public())
		if err != nil {
			logger.Fatal(err)
		}

		csr = ssh.MarshalAuthorizedKey(pubkey)
	} else {
		csr, err = csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Fatal(err)
		}
	}

	var result string
	if isSSH {
		result, err = authority.(*authorities.SSHAuthority).Sign(string(csr), false, lifespan, commonname, []string{"root"})
	} else {
		result, err = authority.(*authorities.TLSAuthority).Sign(string(csr), false, lifespan, commonname, nil, organizations)
	}
	if err != nil {
		logger.Fatal(err)
	}

	err = ioutil.WriteFile(os.Args[5], privkey, os.FileMode(0600))
	if err != nil {
		logger.Fatal(err)
	}
	if isSSH {
		err = ioutil.WriteFile(os.Args[5]+".pub", csr, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	}
	err = ioutil.WriteFile(os.Args[6], []byte(result), os.FileMode(0644))
	if err != nil {
		logger.Fatal(err)
	}
}
