package main

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/util/certutil"
	"github.com/sipb/homeworld/platform/util/csrutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func main() {
	logger := log.New(os.Stderr, "[keylocalcert] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 9 {
		logger.Fatal("usage: keylocalcert <ca-key> <ca-cert> <principal> <lifespan> <out-key> <out-cert> <dnsnames> <organizations>\n  generates a kubernetes or etcd certificate directly")
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
	var dnsnames []string
	if len(os.Args[7]) > 0 {
		dnsnames = strings.Split(os.Args[7], ",")
	}
	var organizations []string
	if len(os.Args[8]) > 0 {
		organizations = strings.Split(os.Args[8], ",")
	}

	// smaller key sizes are okay, because these are limited to a short period
	pkey, privkey, err := certutil.GenerateRSA(2048)
	if err != nil {
		logger.Fatal(err)
	}

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
		result, err = authority.(*authorities.TLSAuthority).Sign(string(csr), len(dnsnames) > 0, lifespan, commonname, dnsnames, organizations)
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
