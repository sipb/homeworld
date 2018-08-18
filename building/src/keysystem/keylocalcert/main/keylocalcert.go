package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"keysystem/keyserver/authorities"
	"log"
	"os"
	"time"
	"util/csrutil"
)

func main() {
	logger := log.New(os.Stderr, "[keylocalcert] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 7 {
		logger.Fatal("usage: keylocalcert <ca-key> <ca-cert> <principal> <lifespan> <out-key> <out-cert>\n  generates a kubernetes or etcd certificate directly")
	}
	keydata, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	certdata, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		logger.Fatal(err)
	}
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		logger.Fatal(err)
	}
	commonname := os.Args[3]
	lifespan, err := time.ParseDuration(os.Args[4])
	if err != nil {
		logger.Fatal(err)
	}

	pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
	if err != nil {
		logger.Fatal(err)
	}
	privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
	csr, err := csrutil.BuildTLSCSR(privkey)
	if err != nil {
		logger.Fatal(err)
	}

	result, err := authority.(*authorities.TLSAuthority).Sign(string(csr), false, lifespan, commonname, nil)
	if err != nil {
		logger.Fatal(err)
	}

	err = ioutil.WriteFile(os.Args[5], privkey, os.FileMode(0600))
	if err != nil {
		logger.Fatal(err)
	}
	err = ioutil.WriteFile(os.Args[6], []byte(result), os.FileMode(0644))
	if err != nil {
		logger.Fatal(err)
	}
}
